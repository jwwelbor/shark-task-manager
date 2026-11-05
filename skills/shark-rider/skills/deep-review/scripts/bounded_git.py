"""Run review commands with a deadline and bounded captured output."""
from contextlib import ExitStack
import math
import os
import signal
import subprocess
import sys
import tempfile
import threading
import time

MAX_OUTPUT_BYTES = 128 * 1024 * 1024


class OutputLimitExceeded(ValueError):
    """A command exceeded the maximum captured output size."""


def run(argv, *, timeout=None, max_output_bytes=MAX_OUTPUT_BYTES, **kwargs):
    timeout = float(os.environ.get('SHARK_REVIEW_GIT_TIMEOUT', '120')) if timeout is None else timeout
    if not math.isfinite(timeout) or timeout <= 0:
        raise ValueError('SHARK_REVIEW_GIT_TIMEOUT must be a positive finite number')
    if not isinstance(max_output_bytes, int) or max_output_bytes <= 0:
        raise ValueError('output limit must be a positive integer')
    check = kwargs.pop('check', False)
    input_data = kwargs.pop('input', None)
    if kwargs.pop('capture_output', False):
        if 'stdout' in kwargs or 'stderr' in kwargs:
            raise ValueError('capture_output cannot be combined with stdout or stderr')
        kwargs['stdout'] = kwargs['stderr'] = subprocess.PIPE
    text_mode = bool(kwargs.get('text') or kwargs.get('universal_newlines') or kwargs.get('encoding') or kwargs.get('errors'))
    with ExitStack() as files:
        if input_data is not None:
            if 'stdin' in kwargs:
                raise ValueError('input cannot be combined with stdin')
            options = {'mode': 'w+t', 'encoding': kwargs.get('encoding'), 'errors': kwargs.get('errors')} if text_mode else {'mode': 'w+b'}
            stream = files.enter_context(tempfile.TemporaryFile(**options))
            stream.write(input_data)
            stream.flush()
            stream.seek(0)
            kwargs['stdin'] = stream
        if os.name != 'nt':
            kwargs['start_new_session'] = True
        proc = subprocess.Popen(argv, **kwargs)
        legacy_windows = None
        output = {}
        exceeded = threading.Event()
        stopping = threading.Event()
        read_errors = []
        def read_pipe(name, stream):
            data = bytearray()
            try:
                while not stopping.is_set():
                    try:
                        chunk = legacy_windows.read(stream.fileno(), 65536) if legacy_windows else os.read(stream.fileno(), 65536)
                    except BlockingIOError:
                        stopping.wait(0.01)
                        continue
                    if not chunk:
                        break
                    if len(data) + len(chunk) > max_output_bytes:
                        exceeded.set()
                        break
                    data.extend(chunk)
            except Exception as exc:
                read_errors.append(exc)
                exceeded.set()
            finally:
                output[name] = bytes(data)
                try:
                    stream.close()
                except Exception as exc:
                    read_errors.append(exc)
                    exceeded.set()
        threads = [threading.Thread(target=read_pipe, args=(name, stream), name='bounded-git-reader', daemon=True)
                   for name, stream in (('stdout', proc.stdout), ('stderr', proc.stderr)) if stream is not None]
        deadline = time.monotonic() + timeout
        try:
            try:
                if os.name == 'nt' and sys.version_info < (3, 12):
                    from windows_pipe import WindowsPipe
                    legacy_windows = WindowsPipe()
                for stream in (proc.stdout, proc.stderr):
                    if stream is not None:
                        if legacy_windows:
                            legacy_windows.set_nonblocking(stream.fileno())
                        else:
                            os.set_blocking(stream.fileno(), False)
            except (AttributeError, ImportError) as exc:
                raise OSError('nonblocking subprocess pipe support is unavailable') from exc
            for thread in threads:
                thread.start()
            while proc.poll() is None or any(thread.is_alive() for thread in threads):
                if read_errors:
                    raise OSError('failed to read captured command output') from read_errors[0]
                if exceeded.is_set():
                    raise OutputLimitExceeded(f'command output exceeds {max_output_bytes} bytes')
                if time.monotonic() >= deadline:
                    raise subprocess.TimeoutExpired(argv, timeout)
                exceeded.wait(min(0.02, max(0, deadline - time.monotonic())))
            if read_errors:
                raise OSError('failed to read captured command output') from read_errors[0]
            if exceeded.is_set():
                raise OutputLimitExceeded(f'command output exceeds {max_output_bytes} bytes')
        except BaseException:
            stopping.set()
            if os.name != 'nt':
                try:
                    os.killpg(proc.pid, signal.SIGKILL)
                except ProcessLookupError:
                    pass
            elif proc.poll() is None:
                proc.kill()
            try:
                proc.wait(timeout=1)
            except subprocess.TimeoutExpired:
                pass
            streams = [stream for stream in (proc.stdout, proc.stderr) if stream is not None]
            for thread, stream in zip(threads, streams):
                if thread.ident is not None:
                    thread.join(timeout=0.2)
                else:
                    stream.close()
            raise
        if text_mode:
            encoding = kwargs.get('encoding') or getattr(proc, 'encoding', None) or 'utf-8'
            errors = kwargs.get('errors') or 'strict'
            output = {key: value.decode(encoding, errors).replace('\r\n', '\n').replace('\r', '\n') for key, value in output.items()}
        result = subprocess.CompletedProcess(argv, proc.returncode, output.get('stdout'), output.get('stderr'))
        if check:
            result.check_returncode()
        return result


def main():
    def terminate(signum, frame):
        raise KeyboardInterrupt()
    signal.signal(signal.SIGTERM, terminate)
    try:
        result = run(['git', '-c', 'core.fsmonitor=false', *sys.argv[1:]], capture_output=True)
        sys.stdout.buffer.write(result.stdout)
        sys.stderr.buffer.write(result.stderr)
        return result.returncode
    except (KeyboardInterrupt, OSError, subprocess.TimeoutExpired, OutputLimitExceeded) as exc:
        print(f'review Git command failed: {exc}', file=sys.stderr)
        return 1


if __name__ == '__main__':
    sys.exit(main())
