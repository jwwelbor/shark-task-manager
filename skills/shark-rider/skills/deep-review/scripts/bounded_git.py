"""Run review Git commands with a deadline and Unix process-group cleanup."""
import math
import os
import signal
import subprocess
import sys


def run(argv, *, timeout=None, **kwargs):
    timeout = float(os.environ.get('SHARK_REVIEW_GIT_TIMEOUT', '120')) if timeout is None else timeout
    if not math.isfinite(timeout) or timeout <= 0:
        raise ValueError('SHARK_REVIEW_GIT_TIMEOUT must be a positive finite number')
    check = kwargs.pop('check', False)
    input_data = kwargs.pop('input', None)
    if input_data is not None:
        kwargs['stdin'] = subprocess.PIPE
    if kwargs.pop('capture_output', False):
        kwargs['stdout'] = subprocess.PIPE
        kwargs['stderr'] = subprocess.PIPE
    if os.name != 'nt':
        kwargs['start_new_session'] = True
    proc = subprocess.Popen(argv, **kwargs)
    try:
        stdout, stderr = proc.communicate(input=input_data, timeout=timeout)
    except BaseException:
        if os.name != 'nt':
            try:
                os.killpg(proc.pid, signal.SIGKILL)
            except ProcessLookupError:
                pass
        else:
            proc.kill()
        proc.communicate()
        raise
    result = subprocess.CompletedProcess(argv, proc.returncode, stdout, stderr)
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
