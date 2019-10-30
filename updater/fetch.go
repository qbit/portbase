package updater

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/google/renameio"

	"github.com/safing/portbase/log"
)

func (reg *ResourceRegistry) fetchFile(rv *ResourceVersion, tries int) error {
	// backoff when retrying
	if tries > 0 {
		time.Sleep(time.Duration(tries*tries) * time.Second)
	}

	// create URL
	downloadURL, err := joinURLandPath(reg.UpdateURLs[tries%len(reg.UpdateURLs)], rv.versionedPath())
	if err != nil {
		return fmt.Errorf("error build url (%s + %s): %s", reg.UpdateURLs[tries%len(reg.UpdateURLs)], rv.versionedPath(), err)
	}

	// check destination dir
	dirPath := filepath.Dir(rv.storagePath())

	err = reg.storageDir.EnsureAbsPath(dirPath)
	if err != nil {
		return fmt.Errorf("could not create updates folder: %s", dirPath)
	}

	// open file for writing
	atomicFile, err := renameio.TempFile(reg.tmpDir.Path, rv.storagePath())
	if err != nil {
		return fmt.Errorf("could not create temp file for download: %s", err)
	}
	defer atomicFile.Cleanup() //nolint:errcheck // ignore error for now, tmp dir will be cleaned later again anyway

	// start file download
	resp, err := http.Get(downloadURL) //nolint:gosec // url is variable on purpose
	if err != nil {
		return fmt.Errorf("error fetching url (%s): %s", downloadURL, err)
	}
	defer resp.Body.Close()

	// download and write file
	n, err := io.Copy(atomicFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed downloading %s: %s", downloadURL, err)
	}
	if resp.ContentLength != n {
		return fmt.Errorf("download unfinished, written %d out of %d bytes", n, resp.ContentLength)
	}

	// finalize file
	err = atomicFile.CloseAtomicallyReplace()
	if err != nil {
		return fmt.Errorf("%s: failed to finalize file %s: %s", reg.Name, rv.storagePath(), err)
	}
	// set permissions
	if !onWindows {
		// TODO: only set executable files to 0755, set other to 0644
		err = os.Chmod(rv.storagePath(), 0755)
		if err != nil {
			log.Warningf("%s: failed to set permissions on downloaded file %s: %s", reg.Name, rv.storagePath(), err)
		}
	}

	log.Infof("%s: fetched %s (stored to %s)", reg.Name, downloadURL, rv.storagePath())
	return nil
}

func (reg *ResourceRegistry) fetchData(downloadPath string, tries int) ([]byte, error) {
	// backoff when retrying
	if tries > 0 {
		time.Sleep(time.Duration(tries*tries) * time.Second)
	}

	// create URL
	downloadURL, err := joinURLandPath(reg.UpdateURLs[tries%len(reg.UpdateURLs)], downloadPath)
	if err != nil {
		return nil, fmt.Errorf("error build url (%s + %s): %s", reg.UpdateURLs[tries%len(reg.UpdateURLs)], downloadPath, err)
	}

	// start file download
	resp, err := http.Get(downloadURL) //nolint:gosec // url is variable on purpose
	if err != nil {
		return nil, fmt.Errorf("error fetching url (%s): %s", downloadURL, err)
	}
	defer resp.Body.Close()

	// download and write file
	buf := bytes.NewBuffer(make([]byte, 0, resp.ContentLength))
	n, err := io.Copy(buf, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed downloading %s: %s", downloadURL, err)
	}
	if resp.ContentLength != n {
		return nil, fmt.Errorf("download unfinished, written %d out of %d bytes", n, resp.ContentLength)
	}

	return buf.Bytes(), nil
}

func joinURLandPath(baseURL, urlPath string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	u.Path = path.Join(u.Path, urlPath)
	return u.String(), nil
}