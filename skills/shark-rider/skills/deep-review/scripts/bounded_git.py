"""Run review Git commands with a deadline and Unix process-group cleanup."""
from contextlib import ExitStack
import math
import os
import signal
import subprocess
import sys
import tempfile


def run(argv, *, timeout=None, **kwargs):
    timeout = float(os.environ.get('SHARK_REVIEW_GIT_TIMEOUT', '120')) if timeout is None else timeout
    if not math.isfinite(timeout) or timeout <= 0:
        raise ValueError('SHARK_REVIEW_GIT_TIMEOUT must be a positive finite number')
    check = kwargs.pop('check', False)
    input_data = kwargs.pop('input', None)
    capture_output = kwargs.pop('capture_output', False)
    if capture_output:
        if 'stdout' in kwargs or 'stderr' in kwargs:
            raise ValueError('capture_output cannot be combined with stdout or stderr')
        kwargs['stdout'] = kwargs['stderr'] = subprocess.PIPE
    text_mode = bool(kwargs.get('text') or kwargs.get('universal_newlines') or kwargs.get('encoding') or kwargs.get('errors'))
    with ExitStack() as files:
        def temporary_stream():
            options = {'mode': 'w+t', 'encoding': kwargs.get('encoding'), 'errors': kwargs.get('errors')} if text_mode else {'mode': 'w+b'}
            return files.enter_context(tempfile.TemporaryFile(**options))
        outputs = {}
        for name in ('stdout', 'stderr'):
            if kwargs.get(name) == subprocess.PIPE:
                outputs[name] = kwargs[name] = temporary_stream()
        if input_data is not None:
            if 'stdin' in kwargs:
                raise ValueError('input cannot be combined with stdin')
            stream = temporary_stream()
            stream.write(input_data)
            stream.flush()
            stream.seek(0)
            kwargs['stdin'] = stream
        if os.name != 'nt':
            kwargs['start_new_session'] = True
        proc = subprocess.Popen(argv, **kwargs)
        try:
            proc.communicate(timeout=timeout)
        except BaseException:
            if os.name != 'nt':
                try:
                    os.killpg(proc.pid, signal.SIGKILL)
                except ProcessLookupError:
                    pass
            else:
                proc.kill()
            try:
                proc.wait(timeout=1)
            except subprocess.TimeoutExpired:
                pass
            raise
        values = {}
        for name, stream in outputs.items():
            stream.seek(0)
            values[name] = stream.read()
        result = subprocess.CompletedProcess(argv, proc.returncode, values.get('stdout'), values.get('stderr'))
        if check:
            result.check_returncode()
        return result



def main():
    def terminate(signum, frame):
        raise KeyboardInterrupt()
    signal.signal(signal.SIGTERM, terminate)
    try:
        result = run(['git', '-c', 'core.fsmonitor=false', *sys.argv[1:]])
        return result.returncode
    except (KeyboardInterrupt, subprocess.TimeoutExpired) as exc:
        print(f'review Git command interrupted or timed out: {exc}', file=sys.stderr)
        return 1


if __name__ == '__main__':
    sys.exit(main())
