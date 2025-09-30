# Ecovacs Firmware Tools

A comprehensive toolkit for working with Ecovacs DEEBOT robot vacuum firmware. This tool allows you to decrypt firmware images and download firmware updates via the official OTA API.

## Features

- **Decrypt Firmware**: Extract and decrypt firmware sections from Ecovacs firmware images
- **Download Firmware**: Automatically discover and download firmware versions via OTA API
- **Manifest Parsing**: Automatically parse and display firmware manifest information
- **Beautiful CLI**: Modern terminal interface with colors and progress bars using Cobra, Logrus, and Lipgloss

## Installation

### From Release

Download the latest binary from the [releases page](https://github.com/denysvitali/ecovacs-firmware-tools/releases).

### From Source

```bash
git clone https://github.com/denysvitali/ecovacs-firmware-tools.git
cd ecovacs-firmware-tools
go build -o ecovacs-firmware-tools .
```

## Usage

### Decrypt Firmware

Decrypt an Ecovacs firmware file and extract all sections:

```bash
# Basic decryption
ecovacs-firmware-tools decrypt firmware.bin

# List sections without decrypting
ecovacs-firmware-tools decrypt -l firmware.bin

# Decrypt with custom parameters
ecovacs-firmware-tools decrypt --device-id 659yh8 --platform px30 firmware.bin

# Decrypt to specific directory
ecovacs-firmware-tools decrypt -o output_dir firmware.bin

# Verbose output
ecovacs-firmware-tools decrypt -v firmware.bin
```

#### Example Output

```
ℹ Loaded firmware: 59611104 bytes
ℹ Device parameters: 659yh8, px30, fw0, v1
✓ Found 6 firmware sections

Manifest: T9AF_px30 v1.4.9 (2021-07-23)

  Section 0: manifest (928 bytes)
  Section 1: pre_upgrade_script (256 bytes)
  Section 2: normal_boot (4.91 MB)
  Section 3: normal_fs (51.86 MB)
  Section 4: mcu (82.34 KB)
  Section 5: post_upgrade_script (16 bytes)
```

### Download Firmware

Search for and download firmware via the official Ecovacs OTA API:

```bash
# Search for firmware (no download)
ecovacs-firmware-tools download --models 659yh8,snxbvc

# Search and download
ecovacs-firmware-tools download --models 659yh8 --download

# Search for specific version
ecovacs-firmware-tools download --models 659yh8 --base-version 1.4.8

# Use specific OTA server
ecovacs-firmware-tools download --server portal-eu.ecouser.net --models 659yh8

# Download to specific directory
ecovacs-firmware-tools download --models 659yh8 --download -o firmware_dir
```

#### Supported Models

- `659yh8` - DEEBOT T9 AIVI
- `snxbvc` - DEEBOT N8 PRO
- Add more models by using their model ID

## Firmware Structure

Ecovacs firmware files contain multiple encrypted sections:

1. **Manifest** - JSON metadata containing firmware version, hardware version, and section information
2. **Pre-upgrade Script** - Shell script executed before firmware upgrade
3. **Boot Image** - Bootloader/kernel image
4. **Filesystem** - Root filesystem (usually SquashFS)
5. **MCU Firmware** - Microcontroller firmware
6. **Post-upgrade Script** - Shell script executed after firmware upgrade

Each section is encrypted using AES-128-CBC with a key derived from the section type and size.

## Development

### Building

```bash
go build -o ecovacs-firmware-tools .
```

### Dependencies

- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [Logrus](https://github.com/sirupsen/logrus) - Structured logging
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Terminal styling
- [Progressbar](https://github.com/schollz/progressbar) - Progress bars

## License

MIT License - Copyright (c) 2025 Denys Vitali

See [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Disclaimer

This tool is for educational and research purposes only. Use at your own risk. The author is not responsible for any damage caused by using this tool.
