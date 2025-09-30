package cmd

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/denysvitali/ecovacs-firmware-tools/pkg/decrypt"
)

type decryptOptions struct {
	deviceID     string
	platform     string
	param        string
	version      int
	outputDir    string
	listSections bool
}

func newDecryptCmd() *cobra.Command {
	opts := &decryptOptions{
		deviceID:  "659yh8",
		platform:  "px30",
		param:     "fw0",
		version:   1,
		outputDir: "decrypted",
	}

	cmd := &cobra.Command{
		Use:   "decrypt <firmware_file>",
		Short: "Decrypt Ecovacs firmware images",
		Long: `Decrypt Ecovacs DEEBOT firmware images.

This command decrypts encrypted firmware sections and extracts the manifest.
The decrypted files are saved to the output directory.

Examples:
  # Decrypt firmware with default parameters
  ecovacs-firmware-tools decrypt firmware.bin

  # Decrypt with custom device parameters
  ecovacs-firmware-tools decrypt --device-id 659yh8 --platform px30 firmware.bin

  # List sections without decrypting
  ecovacs-firmware-tools decrypt --list-sections firmware.bin`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			runDecrypt(opts, args)
		},
	}

	cmd.Flags().StringVarP(&opts.deviceID, "device-id", "d", opts.deviceID, "Device ID")
	cmd.Flags().StringVarP(&opts.platform, "platform", "p", opts.platform, "Platform")
	cmd.Flags().StringVar(&opts.param, "param", opts.param, "Parameter")
	cmd.Flags().IntVar(&opts.version, "version", opts.version, "Version")
	cmd.Flags().StringVarP(&opts.outputDir, "output", "o", opts.outputDir, "Output directory")
	cmd.Flags().BoolVarP(&opts.listSections, "list-sections", "l", false, "Only list firmware sections without decrypting")

	return cmd
}

func runDecrypt(opts *decryptOptions, args []string) {
	firmwareFile := args[0]

	log.WithFields(log.Fields{
		"file":      firmwareFile,
		"device_id": opts.deviceID,
		"platform":  opts.platform,
		"param":     opts.param,
		"version":   opts.version,
	}).Debug("Initializing decryptor")

	// Create decryptor
	decryptor, err := decrypt.NewDecryptor(firmwareFile, opts.deviceID, opts.platform, opts.param, opts.version)
	if err != nil {
		exitWithError("Failed to initialize decryptor: %v", err)
	}

	fmt.Println(renderInfo(fmt.Sprintf("Loaded firmware: %d bytes", len(decryptor.FirmwareData))))
	fmt.Println(renderInfo(fmt.Sprintf("Device parameters: %s, %s, %s, v%d", opts.deviceID, opts.platform, opts.param, opts.version)))

	// Find sections
	sections, err := decryptor.FindSections()
	if err != nil {
		exitWithError("Failed to find firmware sections: %v", err)
	}

	if len(sections) == 0 {
		exitWithError("No valid firmware sections found")
	}

	fmt.Println(renderSuccess(fmt.Sprintf("Found %d firmware sections", len(sections))))
	fmt.Println()

	// Try to decrypt manifest first to get section names
	var manifest *decrypt.Manifest
	sectionNameMap := make(map[uint16]string)

	for _, section := range sections {
		if section.Unkn2 == 3060 { // Manifest section
			sectionType := int(section.Unkn2 >> 12)
			encryptedData := decryptor.FirmwareData[section.DataOffset : section.DataOffset+section.Size]

			key, iv, err := decryptor.DeriveKey(sectionType, section.Size)
			if err == nil {
				if decrypted, err := decryptor.DecryptSection(encryptedData, key, iv); err == nil {
					manifest, _ = decryptor.ParseManifest(decrypted)
					if manifest != nil {
						// Map sections by order: section[0] = manifest, section[i+1] = manifest.sections[i]
						sectionNameMap[sections[0].Unkn2] = "manifest"
						for i := 0; i < len(manifest.Sections) && i+1 < len(sections); i++ {
							sectionNameMap[sections[i+1].Unkn2] = manifest.Sections[i].Name
						}
					}
				}
			}
			break
		}
	}

	// Display manifest info if available
	if manifest != nil {
		fmt.Println(infoStyle.Render(fmt.Sprintf("Manifest: %s v%s (%s)", manifest.Product, manifest.FwVer, manifest.ReleaseDate)))
		fmt.Println()
	}

	// Display sections
	for i, section := range sections {
		sectionName := getSectionNameFromManifest(section.Unkn2, sectionNameMap)
		fmt.Printf("  Section %d: %s (%s)\n",
			i, infoStyle.Render(sectionName),
			formatBytes(section.Size))

		if verbose {
			fmt.Printf("    unkn1=%d, type=%d, unkn2=%d\n", section.Unkn1, section.Type, section.Unkn2)
			fmt.Printf("    Checksum: %s\n", debugStyle.Render(section.Checksum))
		}
	}

	if opts.listSections {
		return
	}

	// Decrypt all sections
	fmt.Println()
	fmt.Println(renderInfo(fmt.Sprintf("Decrypting firmware to: %s", opts.outputDir)))
	fmt.Println()

	results, err := decryptor.DecryptAll(opts.outputDir)
	if err != nil {
		exitWithError("Decryption failed: %v", err)
	}

	// Display results
	successCount := 0
	for i, result := range results {
		sectionName := getSectionNameFromManifest(result.Section.Unkn2, sectionNameMap)

		checksumStatus := renderSuccess("checksum valid")
		if !result.ChecksumValid {
			checksumStatus = renderWarning("checksum mismatch")
		}

		fmt.Printf("  Section %d (%s): %s - %s\n",
			i, sectionName, renderSuccess("decrypted"), checksumStatus)
		fmt.Printf("    %s %s\n", debugStyle.Render("→"), result.OutputFilename)

		if result.Manifest != nil {
			fmt.Printf("    %s Manifest: %s v%s (%s)\n",
				debugStyle.Render("→"),
				result.Manifest.Product,
				result.Manifest.FwVer,
				result.Manifest.ReleaseDate)
		}

		successCount++
	}

	fmt.Println()
	fmt.Println(renderSuccess(fmt.Sprintf("Successfully decrypted %d/%d sections!", successCount, len(sections))))
	fmt.Println(renderInfo(fmt.Sprintf("Decrypted files saved to: %s/", opts.outputDir)))
}

func getSectionNameFromManifest(unkn2 uint16, nameMap map[uint16]string) string {
	// Try to get name from manifest first
	if name, ok := nameMap[unkn2]; ok {
		return name
	}
	return fmt.Sprintf("unknown_%d", unkn2)
}

func formatBytes(bytes int) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d bytes", bytes)
	}
}
