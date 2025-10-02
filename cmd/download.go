package cmd

import (
	"fmt"
	"strings"

	"github.com/denysvitali/ecovacs-firmware-tools/pkg/downloader"
	"github.com/schollz/progressbar/v3"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type downloadOptions struct {
	models       string
	modules      string
	server       string
	downloadFlag bool
	downloadDir  string
	maxVersions  int
	baseVersion  string
}

func newDownloadCmd() *cobra.Command {
	opts := &downloadOptions{
		models:      "659yh8,snxbvc",
		downloadDir: "downloads",
		maxVersions: 50,
	}

	cmd := &cobra.Command{
		Use:   "download",
		Short: "Download Ecovacs firmware via OTA API",
		Long: `Download Ecovacs DEEBOT firmware via OTA API.

This command searches for available firmware versions for specified models
and optionally downloads them.

Examples:
  # Search for firmware (no download)
  ecovacs-firmware-tools download --models 659yh8,snxbvc

  # Search and download firmware
  ecovacs-firmware-tools download --models 659yh8 --download

  # Search for specific version
  ecovacs-firmware-tools download --models 659yh8 --base-version 1.4.8

  # Use specific server
  ecovacs-firmware-tools download --server portal-eu.ecouser.net --models 659yh8`,
		Run: func(cmd *cobra.Command, args []string) {
			runDownload(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.models, "models", "m", opts.models, "Comma-separated model IDs to check")
	cmd.Flags().StringVar(&opts.modules, "modules", "", "Comma-separated firmware modules to check (e.g., 'fw0,mcu,AIConfig')")
	cmd.Flags().StringVar(&opts.server, "server", opts.server, "Override default server")
	cmd.Flags().BoolVar(&opts.downloadFlag, "download", false, "Download found firmware files")
	cmd.Flags().StringVarP(&opts.downloadDir, "output", "o", opts.downloadDir, "Output directory for downloads")
	cmd.Flags().IntVar(&opts.maxVersions, "max-versions", opts.maxVersions, "Maximum versions to check per model")
	cmd.Flags().StringVar(&opts.baseVersion, "base-version", opts.baseVersion, "Base firmware version to check (e.g., '1.2.3')")

	return cmd
}

func runDownload(opts *downloadOptions) {
	fmt.Println(titleStyle.Render("Ecovacs DEEBOT OTA Firmware Downloader"))
	fmt.Println()

	client := downloader.NewClient(opts.server)
	dl := downloader.NewDownloader(opts.downloadDir)

	modelList := strings.Split(opts.models, ",")
	for i := range modelList {
		modelList[i] = strings.TrimSpace(modelList[i])
	}

	fmt.Println(renderInfo(fmt.Sprintf("Server: %s", client.Server)))
	fmt.Println(renderInfo(fmt.Sprintf("Models: %s", strings.Join(modelList, ", "))))
	fmt.Println(renderInfo(fmt.Sprintf("Download: %v", opts.downloadFlag)))
	if opts.baseVersion != "" {
		fmt.Println(renderInfo(fmt.Sprintf("Base Version: %s", opts.baseVersion)))
	}
	fmt.Println()

	foundFirmware := make(map[string]map[string]*downloader.FirmwareInfo)

	for _, model := range modelList {
		modelConfig, exists := downloader.DefaultModels[model]
		if !exists {
			log.Warnf("Unknown model: %s - using default configuration", model)
			modelConfig = downloader.ModelConfig{
				ModelID:         model,
				Name:            fmt.Sprintf("Unknown Model %s", model),
				FirmwareModules: []string{"fw0", "AIConfig"},
			}
		}

		// Override firmware modules if specified by user
		if opts.modules != "" {
			moduleList := strings.Split(opts.modules, ",")
			for i := range moduleList {
				moduleList[i] = strings.TrimSpace(moduleList[i])
			}
			modelConfig.FirmwareModules = moduleList
		}

		fmt.Println(infoStyle.Render(fmt.Sprintf("Checking %s (%s)", modelConfig.Name, model)))

		foundFirmware[model] = make(map[string]*downloader.FirmwareInfo)
		versions := downloader.GenerateVersionPatterns(opts.baseVersion, opts.maxVersions)

		bar := progressbar.NewOptions(len(versions),
			progressbar.OptionSetDescription(fmt.Sprintf("Scanning versions for %s", model)),
			progressbar.OptionSetWidth(40),
			progressbar.OptionShowCount(),
			progressbar.OptionSetTheme(progressbar.Theme{
				Saucer:        "=",
				SaucerHead:    ">",
				SaucerPadding: " ",
				BarStart:      "[",
				BarEnd:        "]",
			}),
		)

		for _, version := range versions {
			for _, module := range modelConfig.FirmwareModules {
				firmware, err := client.CheckFirmware(model, version, module)
				if err != nil {
					log.Debugf("Error checking %s v%s %s: %v", model, version, module, err)
				}
				if firmware != nil {
					key := fmt.Sprintf("%s_v%s", module, firmware.Version)
					foundFirmware[model][key] = firmware
					fmt.Printf("\n%s\n", renderSuccess(fmt.Sprintf("Found %s %s v%s (%d bytes)", model, module, firmware.Version, firmware.Size)))
				}
			}
			bar.Add(1)
		}

		fmt.Println()
	}

	// Display results table
	fmt.Println()
	fmt.Println(titleStyle.Render("Found Firmware"))
	fmt.Println()

	totalFound := 0
	for model, firmwares := range foundFirmware {
		modelName := model
		if config, exists := downloader.DefaultModels[model]; exists {
			modelName = config.Name
		}

		if len(firmwares) == 0 {
			fmt.Printf("  %s (%s): %s\n", modelName, model, renderWarning("No firmware found"))
		} else {
			fmt.Printf("  %s (%s):\n", infoStyle.Render(modelName), model)
			for key, firmware := range firmwares {
				parts := strings.Split(key, "_v")
				module := parts[0]
				checksumShort := firmware.Checksum
				if len(checksumShort) > 12 {
					checksumShort = checksumShort[:12] + "..."
				}
				fmt.Printf("    %s %s v%s - %s bytes (checksum: %s)\n",
					checkmarkStyle.Render("✓"),
					module,
					firmware.Version,
					debugStyle.Render(fmt.Sprintf("%d", firmware.Size)),
					debugStyle.Render(checksumShort))
				totalFound++
			}
		}
	}

	fmt.Println()
	fmt.Println(renderSuccess(fmt.Sprintf("Total firmware files found: %d", totalFound)))

	// Download if requested
	if opts.downloadFlag && totalFound > 0 {
		fmt.Println()
		fmt.Println(renderInfo("Starting downloads..."))
		fmt.Println()

		downloadedCount := 0
		for model, firmwares := range foundFirmware {
			for key, firmware := range firmwares {
				parts := strings.Split(key, "_v")
				module := parts[0]

				filename := dl.GetFilename(model, module, firmware)
				fmt.Printf("Downloading %s...\n", filename)

				bar := progressbar.DefaultBytes(
					firmware.Size,
					"",
				)

				filepath, err := dl.DownloadFirmware(model, module, firmware, func(progress downloader.DownloadProgress) {
					bar.Set64(progress.BytesDownloaded)
				})

				if err != nil {
					fmt.Println(renderError(fmt.Sprintf("Failed to download: %v", err)))
					continue
				}

				fmt.Printf("%s\n", renderSuccess(fmt.Sprintf("Downloaded to: %s", filepath)))
				downloadedCount++
			}
		}

		fmt.Println()
		fmt.Println(renderSuccess(fmt.Sprintf("Successfully downloaded %d firmware files to %s/", downloadedCount, opts.downloadDir)))
	}
}