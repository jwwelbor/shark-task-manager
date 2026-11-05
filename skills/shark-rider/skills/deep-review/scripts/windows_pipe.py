"""Nonblocking anonymous-pipe reads on Windows Python before 3.12."""
import ctypes
from ctypes import wintypes
import msvcrt

ERROR_BROKEN_PIPE = 109
ERROR_NO_DATA = 232
PIPE_NOWAIT = 1


class WindowsPipe:
    def __init__(self):
        self.kernel = ctypes.WinDLL('kernel32', use_last_error=True)
        self.kernel.SetNamedPipeHandleState.argtypes = [wintypes.HANDLE, ctypes.POINTER(wintypes.DWORD), ctypes.c_void_p, ctypes.c_void_p]
        self.kernel.SetNamedPipeHandleState.restype = wintypes.BOOL
        self.kernel.ReadFile.argtypes = [wintypes.HANDLE, ctypes.c_void_p, wintypes.DWORD, ctypes.POINTER(wintypes.DWORD), ctypes.c_void_p]
        self.kernel.ReadFile.restype = wintypes.BOOL

    def set_nonblocking(self, fd):
        mode = wintypes.DWORD(PIPE_NOWAIT)
        if not self.kernel.SetNamedPipeHandleState(msvcrt.get_osfhandle(fd), ctypes.byref(mode), None, None):
            raise ctypes.WinError(ctypes.get_last_error())

    def read(self, fd, size):
        buffer = ctypes.create_string_buffer(size)
        count = wintypes.DWORD()
        if self.kernel.ReadFile(msvcrt.get_osfhandle(fd), buffer, size, ctypes.byref(count), None):
            return buffer.raw[:count.value]
        code = ctypes.get_last_error()
        if code == ERROR_NO_DATA:
            raise BlockingIOError('nonblocking pipe has no data')
        if code == ERROR_BROKEN_PIPE:
            return b''
        raise ctypes.WinError(code)
