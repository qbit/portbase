package osdetail

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
)

const powershellGetFileDescription = `Get-ItemProperty "%s" | Format-List`

// GetBinaryNameFromSystem queries the operating system for a human readable
// name for the given binary path.
func GetBinaryNameFromSystem(path string) (string, error) {
	// Get FileProperties via Powershell call.
	output, err := runPowershellCmd(fmt.Sprintf(powershellGetFileDescription, path))
	if err != nil {
		return "", fmt.Errorf("failed to get file properties of %s: %s", path, err)
	}

	// Create scanner for the output.
	scanner := bufio.NewScanner(bytes.NewBufferString(output))
	scanner.Split(bufio.ScanLines)

	// Search for the FileDescription line.
	for scanner.Scan() {
		// Split line up into fields.
		fields := strings.Fields(scanner.Text())
		// Discard lines with less than two fields.
		if len(fields) < 2 {
			continue
		}
		// Skip all lines that we aren't looking for.
		if fields[0] != "FileDescription:" {
			continue
		}

		// Clean and return.
		return cleanFileDescription(fields[1:]), nil
	}

	// Generate a default name as default.
	return "", ErrNotFound
}

const powershellGetIcon = `Add-Type -AssemblyName System.Drawing
$Icon = [System.Drawing.Icon]::ExtractAssociatedIcon("%s")
$MemoryStream = New-Object System.IO.MemoryStream
$Icon.save($MemoryStream)
$Bytes = $MemoryStream.ToArray()
$MemoryStream.Flush()
$MemoryStream.Dispose()
[convert]::ToBase64String($Bytes)`

// TODO: This returns a small and crappy icon.

// Saving a better icon to file works:
/*
Add-Type -AssemblyName System.Drawing
$ImgList = New-Object System.Windows.Forms.ImageList
$ImgList.ImageSize = New-Object System.Drawing.Size(256,256)
$ImgList.ColorDepth = 32
$Icon = [System.Drawing.Icon]::ExtractAssociatedIcon("C:\Program Files (x86)\Mozilla Firefox\firefox.exe")
$ImgList.Images.Add($Icon);
$BigIcon = $ImgList.Images.Item(0)
$BigIcon.Save("test.png")
*/

// But not saving to a memory stream:
/*
Add-Type -AssemblyName System.Drawing
$ImgList = New-Object System.Windows.Forms.ImageList
$ImgList.ImageSize = New-Object System.Drawing.Size(256,256)
$ImgList.ColorDepth = 32
$Icon = [System.Drawing.Icon]::ExtractAssociatedIcon("C:\Program Files (x86)\Mozilla Firefox\firefox.exe")
$ImgList.Images.Add($Icon);
$MemoryStream = New-Object System.IO.MemoryStream
$BigIcon = $ImgList.Images.Item(0)
$BigIcon.Save($MemoryStream)
$Bytes = $MemoryStream.ToArray()
$MemoryStream.Flush()
$MemoryStream.Dispose()
[convert]::ToBase64String($Bytes)
*/

// GetBinaryIconFromSystem queries the operating system for the associated icon
// for a given binary path.
func GetBinaryIconFromSystem(path string) (string, error) {
	// Get Associated File Icon via Powershell call.
	output, err := runPowershellCmd(fmt.Sprintf(powershellGetIcon, path))
	if err != nil {
		return "", fmt.Errorf("failed to get file properties of %s: %s", path, err)
	}

	return "data:image/png;base64," + output, nil
}
