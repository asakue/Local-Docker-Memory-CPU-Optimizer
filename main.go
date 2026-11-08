package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"dockertrim/internal/analyzer"
	"dockertrim/internal/cli"
	"dockertrim/internal/docker"
	"dockertrim/internal/logger"
	"dockertrim/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/shirou/gopsutil/v3/mem"
)

func waitBeforeExit(code int) {
	// If executed from Explorer (stdin is terminal or interactive), give user a chance to see message
	if fi, err := os.Stdin.Stat(); err == nil && (fi.Mode()&os.ModeCharDevice) != 0 {
		fmt.Println("\nНажмите Enter для выхода...")
		bufio.NewReader(os.Stdin).ReadBytes('\n')
	}
	os.Exit(code)
}

func main() {
	summaryFlag := flag.Bool("summary", false, "Вывести краткий текстовый отчет анализа без запуска TUI")
	flag.BoolVar(summaryFlag, "s", false, "Вывести краткий текстовый отчет (сокращение для --summary)")

	jsonFlag := flag.Bool("json", false, "Вывести отчет анализа в формате JSON")
	flag.BoolVar(jsonFlag, "j", false, "Вывести отчет в JSON (сокращение для --json)")

	demoFlag := flag.Bool("demo", false, "Демонстрационный режим с тестовыми данными (без реального Docker)")
	flag.BoolVar(demoFlag, "d", false, "Демо-режим (сокращение для --demo)")

	idleHours := flag.Float64("idle-hours", 2.0, "Порог времени в часах для детекта простаивающих контейнеров")
	idleCPU := flag.Float64("idle-cpu", 0.1, "Порог CPU% для детекта простаивающих контейнеров")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "🐳 dockertrim — CLI/TUI-утилита для ресурсной гигиены Docker\n\n")
		fmt.Fprintf(os.Stderr, "Использование:\n")
		fmt.Fprintf(os.Stderr, "  dockertrim [опции]\n\n")
		fmt.Fprintf(os.Stderr, "Опции:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nГорячие клавиши в TUI:\n")
		fmt.Fprintf(os.Stderr, "  Tab / 1, 2      Переключение между вкладками 'Контейнеры' и 'Диск'\n")
		fmt.Fprintf(os.Stderr, "  ↑ / ↓ / j / k   Навигация по списку\n")
		fmt.Fprintf(os.Stderr, "  Space / v       Выбрать / снять выбор строки\n")
		fmt.Fprintf(os.Stderr, "  a               Выбрать все / снять со всех\n")
		fmt.Fprintf(os.Stderr, "  s / p / r / x   Stop / Pause / Restart / Remove для выбранных\n")
		fmt.Fprintf(os.Stderr, "  l / i           Просмотр логов / инспекция контейнера\n")
		fmt.Fprintf(os.Stderr, "  m / u / n       Сортировка по RAM, CPU или имени\n")
		fmt.Fprintf(os.Stderr, "  Enter           Запустить очистку на вкладке Диска\n")
		fmt.Fprintf(os.Stderr, "  q / Ctrl+C      Выход\n")
	}

	flag.Parse()

	_ = logger.Init("dockertrim.log")
	defer logger.Close()
	logger.Info("Starting dockertrim execution (summary=%v, json=%v, demo=%v)", *summaryFlag, *jsonFlag, *demoFlag)

	idleCfg := analyzer.IdleConfig{
		CPUThresholdPercent: *idleCPU,
		MinDuration:         time.Duration(*idleHours * float64(time.Hour)),
	}
	oomCfg := analyzer.DefaultOOMConfig()

	// Demo mode without Docker
	if *demoFlag {
		runDemoMode(idleCfg, oomCfg, *summaryFlag, *jsonFlag)
		return
	}

	// Initialize Docker Service
	svc, err := docker.NewDockerService()
	var dockerErr error
	if err != nil {
		dockerErr = err
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		if pingErr := svc.Ping(ctx); pingErr != nil {
			dockerErr = pingErr
		}
		cancel()
	}

	if dockerErr != nil {
		if *summaryFlag || *jsonFlag {
			fmt.Fprintf(os.Stderr, "❌ Docker Daemon не доступен: %v\n", dockerErr)
			fmt.Fprintf(os.Stderr, "💡 Убедитесь, что Docker Desktop запущен (npipe:////./pipe/dockerDesktopLinuxEngine).\n")
			waitBeforeExit(1)
		}

		// When in interactive mode and Docker is offline, automatically offer Demo mode
		fmt.Println("⚠️  Внимание: Docker Daemon в данный момент недоступен.")
		fmt.Printf("   Ошибка: %v\n\n", dockerErr)
		fmt.Println("💡 Запустить dockertrim в ДЕМОНСТРАЦИОННОМ режиме с тестовыми контейнерами?")
		fmt.Print("   Нажмите [Y] / [Enter] для Демо-режима или любую другую клавишу для выхода: ")

		var response string
		fmt.Scanln(&response)
		if response == "" || response == "y" || response == "Y" || response == "yes" || response == "д" || response == "Д" {
			runDemoMode(idleCfg, oomCfg, false, false)
			return
		}
		waitBeforeExit(1)
	}
	defer svc.Close()

	// One-shot CLI modes
	if *summaryFlag || *jsonFlag {
		ctxCollect, cancelCollect := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelCollect()

		var hostTotal, hostUsed uint64
		if v, err := mem.VirtualMemory(); err == nil {
			hostTotal = v.Total
			hostUsed = v.Used
		}

		containers, err := svc.ListContainers(ctxCollect)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Ошибка получения списка контейнеров: %v\n", err)
			waitBeforeExit(1)
		}

		containers = svc.EnrichContainerMetrics(ctxCollect, containers, hostTotal)
		now := time.Now()
		for i := range containers {
			analyzer.AnalyzeIdle(&containers[i], idleCfg, now)
			analyzer.AnalyzeMemory(&containers[i], hostTotal, oomCfg)
		}

		disk, err := svc.AuditDiskSpace(ctxCollect)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Ошибка аудита диска Docker: %v\n", err)
			waitBeforeExit(1)
		}

		report := analyzer.BuildSystemReport(containers, disk, hostTotal, hostUsed, idleCfg, oomCfg)

		if *jsonFlag {
			if err := cli.PrintJSON(report); err != nil {
				fmt.Fprintf(os.Stderr, "❌ Ошибка сериализации JSON: %v\n", err)
				waitBeforeExit(1)
			}
			return
		}

		cli.PrintSummary(report)
		return
	}

	// Interactive TUI Mode
	model := tui.NewModel(svc, idleCfg, oomCfg, false)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Ошибка работы TUI: %v\n", err)
		waitBeforeExit(1)
	}
}

func runDemoMode(idleCfg analyzer.IdleConfig, oomCfg analyzer.OOMConfig, summary, jsonOutput bool) {
	var hostTotal, hostUsed uint64
	if v, err := mem.VirtualMemory(); err == nil {
		hostTotal = v.Total
		hostUsed = v.Used
	} else {
		hostTotal = 16 * 1024 * 1024 * 1024
		hostUsed = 8 * 1024 * 1024 * 1024
	}

	containers, disk := docker.GenerateMockData(idleCfg, oomCfg, hostTotal)
	report := analyzer.BuildSystemReport(containers, disk, hostTotal, hostUsed, idleCfg, oomCfg)

	if jsonOutput {
		if err := cli.PrintJSON(report); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Ошибка JSON: %v\n", err)
			waitBeforeExit(1)
		}
		return
	}

	if summary {
		cli.PrintSummary(report)
		return
	}

	model := tui.NewModel(nil, idleCfg, oomCfg, true)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Ошибка TUI: %v\n", err)
		waitBeforeExit(1)
	}
}
