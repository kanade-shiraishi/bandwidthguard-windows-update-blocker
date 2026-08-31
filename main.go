package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"unsafe"
)

const appName = "BandwidthGuard"
const appAuthor = "Kanade Shiraishi"
const appVersion = "1.0"

type service struct {
	name  string
	label string
}

var targetServices = []service{
	{"DoSvc", "Delivery Optimization"},
	{"BITS", "Background Intelligent Transfer Service"},
	{"wuauserv", "Windows Update"},
	{"WaaSMedicSvc", "Windows Update Medic Service"},
}

var targetTasks = []string{
	`\Microsoft\Windows\WindowsUpdate\Scheduled Start`,
	`\Microsoft\Windows\WindowsUpdate\sih`,
	`\Microsoft\Windows\WindowsUpdate\sihboot`,
	`\Microsoft\Windows\UpdateOrchestrator\Schedule Scan`,
	`\Microsoft\Windows\UpdateOrchestrator\Schedule Scan Static Task`,
	`\Microsoft\Windows\UpdateOrchestrator\USO_UxBroker`,
}

const auKeyPath = `HKLM\SOFTWARE\Policies\Microsoft\Windows\WindowsUpdate\AU`
const doKeyPath = `HKLM\SOFTWARE\Policies\Microsoft\Windows\DeliveryOptimization`
const nduKeyPath = `HKLM\SYSTEM\CurrentControlSet\Services\Ndu\Parameters`

func main() {
	lockFlag := flag.Bool("lock", false, "apply bandwidth lockdown")
	unlockFlag := flag.Bool("unlock", false, "restore default Windows behavior")
	statusFlag := flag.Bool("status", false, "show current state")
	flag.Parse()

	if !isElevated() {
		if err := relaunchElevated(); err != nil {
			fmt.Println("Could not get administrator rights:", err)
			fmt.Println("Right-click " + appName + ".exe and choose \"Run as administrator\".")
			pause()
			os.Exit(1)
		}
		os.Exit(0)
	}

	switch {
	case *lockFlag:
		lockdown()
	case *unlockFlag:
		restore()
	case *statusFlag:
		status()
	default:
		menu()
	}
}

func menu() {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Println()
		fmt.Println(appName, appVersion, "-", appAuthor)
		fmt.Println("--------------------------------------")
		fmt.Println("1. Lock down bandwidth")
		fmt.Println("2. Restore defaults")
		fmt.Println("3. Show status")
		fmt.Println("4. Exit")
		fmt.Print("Choose an option: ")

		line, _ := reader.ReadString('\n')
		choice := strings.TrimSpace(line)

		switch choice {
		case "1":
			lockdown()
		case "2":
			restore()
		case "3":
			status()
		case "4":
			return
		default:
			fmt.Println("Not a valid option.")
		}
	}
}

func lockdown() {
	fmt.Println("Locking down bandwidth usage...")
	fmt.Println()

	for _, svc := range targetServices {
		fmt.Printf("Stopping %s (%s)... ", svc.label, svc.name)
		if err := runService("stop", svc.name); err != nil {
			fmt.Println("already stopped or unavailable")
		} else {
			fmt.Println("done")
		}

		fmt.Printf("Disabling %s from starting again... ", svc.name)
		if err := setServiceStart(svc.name, "disabled"); err != nil {
			fmt.Println("failed:", err)
		} else {
			fmt.Println("done")
		}
	}

	fmt.Println()
	for _, task := range targetTasks {
		fmt.Printf("Disabling scheduled task %s... ", task)
		if err := setTaskState(task, false); err != nil {
			fmt.Println("not found or already disabled")
		} else {
			fmt.Println("done")
		}
	}

	fmt.Println()
	fmt.Print("Blocking Windows Update from running automatically... ")
	if err := regSetDword(auKeyPath, "NoAutoUpdate", 1); err != nil {
		fmt.Println("failed:", err)
	} else {
		fmt.Println("done")
	}

	fmt.Print("Turning off Delivery Optimization downloads... ")
	if err := regSetDword(doKeyPath, "DODownloadMode", 0); err != nil {
		fmt.Println("failed:", err)
	} else {
		fmt.Println("done")
	}

	fmt.Print("Marking network connections as metered... ")
	if err := setMeteredDefault(true); err != nil {
		fmt.Println("failed:", err)
	} else {
		fmt.Println("done")
	}

	fmt.Println()
	fmt.Println("Lockdown complete. Background downloads and update traffic are blocked.")
	if flag_wasMenu() {
		pause()
	}
}

func restore() {
	fmt.Println("Restoring Windows default behavior...")
	fmt.Println()

	for _, svc := range targetServices {
		fmt.Printf("Re-enabling %s (%s)... ", svc.label, svc.name)
		if err := setServiceStart(svc.name, "demand"); err != nil {
			fmt.Println("failed:", err)
		} else {
			fmt.Println("done")
		}

		fmt.Printf("Starting %s... ", svc.name)
		if err := runService("start", svc.name); err != nil {
			fmt.Println("failed:", err)
		} else {
			fmt.Println("done")
		}
	}

	fmt.Println()
	for _, task := range targetTasks {
		fmt.Printf("Re-enabling scheduled task %s... ", task)
		if err := setTaskState(task, true); err != nil {
			fmt.Println("not found")
		} else {
			fmt.Println("done")
		}
	}

	fmt.Println()
	fmt.Print("Removing Windows Update restriction... ")
	if err := regDeleteValue(auKeyPath, "NoAutoUpdate"); err != nil {
		fmt.Println("nothing to remove")
	} else {
		fmt.Println("done")
	}

	fmt.Print("Restoring Delivery Optimization defaults... ")
	if err := regDeleteValue(doKeyPath, "DODownloadMode"); err != nil {
		fmt.Println("nothing to remove")
	} else {
		fmt.Println("done")
	}

	fmt.Print("Clearing metered connection default... ")
	if err := setMeteredDefault(false); err != nil {
		fmt.Println("failed:", err)
	} else {
		fmt.Println("done")
	}

	fmt.Println()
	fmt.Println("Restore complete. Windows Update and Delivery Optimization are back to normal.")
	if flag_wasMenu() {
		pause()
	}
}

func status() {
	fmt.Println(appName, "status")
	fmt.Println("--------------------------------------")

	fmt.Println()
	fmt.Println("Services:")
	for _, svc := range targetServices {
		state, startType := queryService(svc.name)
		fmt.Printf("  %-14s state=%-10s start=%-10s\n", svc.name, state, startType)
	}

	fmt.Println()
	fmt.Println("Scheduled tasks:")
	for _, task := range targetTasks {
		fmt.Printf("  %-55s %s\n", task, queryTaskState(task))
	}

	fmt.Println()
	fmt.Println("Registry policy:")
	fmt.Printf("  NoAutoUpdate     = %s\n", regReadValue(auKeyPath, "NoAutoUpdate"))
	fmt.Printf("  DODownloadMode   = %s\n", regReadValue(doKeyPath, "DODownloadMode"))

	fmt.Println()
	fmt.Println("Metered connection default:")
	fmt.Printf("  %s\n", queryMeteredDefault())

	if flag_wasMenu() {
		pause()
	}
}

func runService(action, name string) error {
	cmd := exec.Command("sc", action, name)
	return cmd.Run()
}

func setServiceStart(name, startType string) error {
	cmd := exec.Command("sc", "config", name, "start="+startType)
	return cmd.Run()
}

func queryService(name string) (state string, startType string) {
	state = "unknown"
	startType = "unknown"

	out, err := exec.Command("sc", "query", name).CombinedOutput()
	if err == nil {
		text := string(out)
		switch {
		case strings.Contains(text, "RUNNING"):
			state = "running"
		case strings.Contains(text, "STOPPED"):
			state = "stopped"
		case strings.Contains(text, "PAUSED"):
			state = "paused"
		}
	} else {
		state = "missing"
	}

	out, err = exec.Command("sc", "qc", name).CombinedOutput()
	if err == nil {
		text := string(out)
		switch {
		case strings.Contains(text, "DISABLED"):
			startType = "disabled"
		case strings.Contains(text, "DEMAND_START"):
			startType = "manual"
		case strings.Contains(text, "AUTO_START"):
			startType = "automatic"
		}
	}

	return state, startType
}

func setTaskState(taskPath string, enable bool) error {
	action := "disable"
	if enable {
		action = "enable"
	}
	cmd := exec.Command("schtasks", "/Change", "/TN", taskPath, "/"+capitalize(action))
	return cmd.Run()
}

func queryTaskState(taskPath string) string {
	out, err := exec.Command("schtasks", "/Query", "/TN", taskPath, "/FO", "LIST").CombinedOutput()
	if err != nil {
		return "not found"
	}
	text := string(out)
	if strings.Contains(text, "Disabled") {
		return "disabled"
	}
	if strings.Contains(text, "Ready") || strings.Contains(text, "Running") {
		return "enabled"
	}
	return "unknown"
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func regSetDword(keyPath, valueName string, value int) error {
	cmd := exec.Command("reg", "add", keyPath, "/v", valueName, "/t", "REG_DWORD", "/d", fmt.Sprint(value), "/f")
	return cmd.Run()
}

func regDeleteValue(keyPath, valueName string) error {
	cmd := exec.Command("reg", "delete", keyPath, "/v", valueName, "/f")
	return cmd.Run()
}

func regReadValue(keyPath, valueName string) string {
	out, err := exec.Command("reg", "query", keyPath, "/v", valueName).CombinedOutput()
	if err != nil {
		return "not set"
	}
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.Contains(line, valueName) {
			fields := strings.Fields(line)
			if len(fields) > 0 {
				return fields[len(fields)-1]
			}
		}
	}
	return "not set"
}

func setMeteredDefault(metered bool) error {
	value := 1
	if !metered {
		value = 2
	}
	return regSetDword(nduKeyPath, "EthernetCostSourceOverride", value)
}

func queryMeteredDefault() string {
	val := regReadValue(nduKeyPath, "EthernetCostSourceOverride")
	switch val {
	case "0x1":
		return "forced metered"
	case "0x2":
		return "forced unmetered (Windows default)"
	default:
		return "not configured (Windows default)"
	}
}

func flag_wasMenu() bool {
	return !hasAnyFlag()
}

func hasAnyFlag() bool {
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "-") {
			return true
		}
	}
	return false
}

func pause() {
	fmt.Print("\nPress Enter to continue...")
	bufio.NewReader(os.Stdin).ReadString('\n')
}

func isElevated() bool {
	var token syscall.Token
	proc, err := syscall.GetCurrentProcess()
	if err != nil {
		return false
	}
	err = syscall.OpenProcessToken(proc, syscall.TOKEN_QUERY, &token)
	if err != nil {
		return false
	}
	defer token.Close()

	var elevation struct {
		TokenIsElevated uint32
	}
	var returnedLen uint32
	err = syscall.GetTokenInformation(
		token,
		syscall.TokenElevation,
		(*byte)(unsafe.Pointer(&elevation)),
		uint32(unsafe.Sizeof(elevation)),
		&returnedLen,
	)
	if err != nil {
		return false
	}
	return elevation.TokenIsElevated != 0
}

func relaunchElevated() error {
	verb := "runas"
	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	args := strings.Join(os.Args[1:], " ")

	verbPtr, _ := syscall.UTF16PtrFromString(verb)
	exePtr, _ := syscall.UTF16PtrFromString(exePath)
	cwd, _ := os.Getwd()
	cwdPtr, _ := syscall.UTF16PtrFromString(cwd)
	argPtr, _ := syscall.UTF16PtrFromString(args)

	shell32 := syscall.NewLazyDLL("shell32.dll")
	shellExecute := shell32.NewProc("ShellExecuteW")

	ret, _, _ := shellExecute.Call(
		0,
		uintptr(unsafe.Pointer(verbPtr)),
		uintptr(unsafe.Pointer(exePtr)),
		uintptr(unsafe.Pointer(argPtr)),
		uintptr(unsafe.Pointer(cwdPtr)),
		1,
	)

	if ret <= 32 {
		return fmt.Errorf("ShellExecute failed with code %d", ret)
	}
	return nil
}
