//go:build windows

package cert

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

const (
	rootCAThumbprint = "1B2350C79FDDB4EAA97527A4155CFF7F0472EAB3"
	rootCAName       = "Lihugang Root CA"
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
// It verifies presence in either LocalMachine or CurrentUser Root stores without triggering
// network revocation checks (which fail on private Root CAs and cause certutil -verifystore to fail).
func IsRootCAInstalled() bool {
	// Method 1: Check via PowerShell against both LocalMachine and CurrentUser Root stores
	psScript := fmt.Sprintf(`[bool](Get-ChildItem Cert:\LocalMachine\Root, Cert:\CurrentUser\Root -ErrorAction SilentlyContinue | Where-Object { $_.Thumbprint -eq '%s' -or $_.Subject -like '*%s*' })`,
		rootCAThumbprint, rootCAName)

	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", psScript)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if out, err := cmd.Output(); err == nil {
		res := strings.TrimSpace(string(out))
		if strings.EqualFold(res, "True") {
			return true
		}
		if strings.EqualFold(res, "False") {
			return false
		}
	}

	// Method 2 (Fallback): Check via certutil -store (NOT -verifystore)
	// Check CurrentUser Root store
	cmdUser := exec.Command("certutil", "-store", "-user", "Root", rootCAThumbprint)
	cmdUser.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmdUser.Run(); err == nil {
		return true
	}

	// Check LocalMachine Root store
	cmdMachine := exec.Command("certutil", "-store", "Root", rootCAThumbprint)
	cmdMachine.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmdMachine.Run(); err == nil {
		return true
	}

	// Also check by Common Name
	cmdCN := exec.Command("certutil", "-store", "-user", "Root", rootCAName)
	cmdCN.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmdCN.Run(); err == nil {
		return true
	}

	cmdCNMachine := exec.Command("certutil", "-store", "Root", rootCAName)
	cmdCNMachine.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmdCNMachine.Run(); err == nil {
		return true
	}

	return false
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
	// Install to both LocalMachine Root and CurrentUser Root plus TrustedPublisher
	batchFile := filepath.Join(tempDir, "install_lihugang_ca.cmd")
	batchContent := fmt.Sprintf(`@echo off
certutil -addstore -f "Root" "%s" >nul 2>&1
certutil -user -addstore -f "Root" "%s" >nul 2>&1
certutil -addstore -f "TrustedPublisher" "%s" >nul 2>&1
`, certFile, certFile, certFile)

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
		return nil
	}

	logger.Warnf("[DeskNotify] 提示: 证书导入未完成或已取消")
	return fmt.Errorf("证书导入未完成或已取消")
}
