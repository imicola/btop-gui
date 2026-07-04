//go:build linux && cgo

package processctl

/*
#include <errno.h>
#include <stdlib.h>
#include <unistd.h>

static int fork_exec_sleep(const char *duration) {
    pid_t pid = fork();
    if (pid < 0) {
        return -errno;
    }
    if (pid == 0) {
        char *const argv[] = {(char *)"sleep", (char *)duration, NULL};
        execv("/bin/sleep", argv);
        _exit(127);
    }
    return (int)pid;
}
*/
import "C"

import (
	"fmt"
	"strconv"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

// ForkSleep 调用 libc fork()，子进程随即 execv /bin/sleep。
// fork 后的子进程只执行 async-signal-safe 的 execv/_exit，避免触碰 Go runtime。
func ForkSleep(seconds int) (int, error) {
	duration := C.CString(strconv.Itoa(seconds))
	defer C.free(unsafe.Pointer(duration))
	pid := int(C.fork_exec_sleep(duration))
	if pid < 0 {
		return 0, fmt.Errorf("fork: %w", syscall.Errno(-pid))
	}
	go func() {
		var status unix.WaitStatus
		_, _ = unix.Wait4(pid, &status, 0, nil)
	}()
	return pid, nil
}
