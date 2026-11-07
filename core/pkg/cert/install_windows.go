//go:build windows

package cert

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"

	"github.com/lihugang/desk-notify/core/pkg/logger"
)

var (
	shell32            = syscall.NewLazyDLL("shell32.dll")
	procShellExecuteEx = shell32.NewProc("ShellExecuteExW")
	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	procWaitForSingle  = kernel32.NewProc("WaitForSingleObject")
	procCloseHandle    = kernel32.NewProc("CloseHandle")
)

const (
	SW_HIDE                 = 0
	SEE_MASK_NOCLOSEPROCESS = 0x00000040
	INFINITE                = 0xFFFFFFFF
)

type shellExecuteInfo struct {
	cbSize         uint32
	fMask          uint32
	hwnd           uintptr
	lpVerb         *uint16
	lpFile         *uint16
	lpParameters   *uint16
	lpDirectory    *uint16
	nShow          int32
	hInstApp       uintptr
	lpIDList       uintptr
	lpClass        *uint16
	hkeyClass      uintptr
	dwHotKey       uint32
	hIcon          uintptr
	hProcess       uintptr
}

// IsRootCAInstalled checks if the Lihugang Root CA is already installed in Windows cert stores.
func IsRootCAInstalled() bool {
	// Check Root store
	cmdRoot := exec.Command("certutil", "-verifystore", "Root", "Lihugang Root CA")
	cmdRoot.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmdRoot.Run(); err != nil {
		return false
	}

	// Check TrustedPublisher store
	cmdPub := exec.Command("certutil", "-verifystore", "TrustedPublisher", "Lihugang Root CA")
	cmdPub.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmdPub.Run(); err != nil {
		return false
	}

	return true
}

// EnsureRootCAInstalled automatically prompts for UAC elevation to install
// the embedded Lihugang Root CA into the Windows certificate store if not already present.
func EnsureRootCAInstalled() error {
	if IsRootCAInstalled() {
		return nil
	}

	logger.Infof("[DeskNotify] 检测到系统尚未安装作者统一根证书，正在请求管理员提权安装...")

	// 1. Write embedded Root CA certificate to temp file
	tempDir := os.TempDir()
	certFile := filepath.Join(tempDir, "Lihugang-CA.cer")
	if err := os.WriteFile(certFile, GetRootCACertDER(), 0644); err != nil {
		return fmt.Errorf("failed to write temp cert file: %w", err)
	}
	defer os.Remove(certFile)

	// 2. Create elevated install batch script in temp dir
	batchFile := filepath.Join(tempDir, "install_lihugang_ca.cmd")
	batchContent := fmt.Sprintf(`@echo off
certutil -addstore -f "Root" "%s" >nul 2>&1
certutil -addstore -f "TrustedPublisher" "%s" >nul 2>&1
`, certFile, certFile)

	if err := os.WriteFile(batchFile, []byte(batchContent), 0644); err != nil {
		return fmt.Errorf("failed to write temp batch file: %w", err)
	}
	defer os.Remove(batchFile)

	// 3. Prepare ShellExecuteExW with verb "runas" for UAC prompt
	verbPtr, _ := syscall.UTF16PtrFromString("runas")
	filePtr, _ := syscall.UTF16PtrFromString("cmd.exe")
	paramsPtr, _ := syscall.UTF16PtrFromString(fmt.Sprintf(`/c "%s"`, batchFile))
	dirPtr, _ := syscall.UTF16PtrFromString(tempDir)

	var sei shellExecuteInfo
	sei.cbSize = uint32(unsafe.Sizeof(sei))
	sei.fMask = SEE_MASK_NOCLOSEPROCESS
	sei.lpVerb = verbPtr
	sei.lpFile = filePtr
	sei.lpParameters = paramsPtr
	sei.lpDirectory = dirPtr
	sei.nShow = SW_HIDE

	ret, _, err := procShellExecuteEx.Call(uintptr(unsafe.Pointer(&sei)))
	if ret == 0 {
		return fmt.Errorf("提权请求被用户取消或失败: %v", err)
	}

	// 4. Wait for the elevated installation to complete
	if sei.hProcess != 0 {
		procWaitForSingle.Call(sei.hProcess, uintptr(INFINITE))
		procCloseHandle.Call(sei.hProcess)
	}

	if IsRootCAInstalled() {
		logger.Infof("[DeskNotify] 成功: Lihugang 个人根证书已成功导入 Windows 受信任证书区！")
	} else {
		logger.Warnf("[DeskNotify] 提示: 证书导入未完成或已取消")
	}

	return nil
}
