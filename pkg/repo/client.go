/*
 * MIT License
 *
 * Copyright (c) since 2021,  flomesh.io Authors.
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package repo

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"

	"github.com/flomesh-io/fsm/pkg/constants"
)

type PipyRepoClient struct {
	baseUrl          string
	defaultTransport *http.Transport
	httpClient       *resty.Client
	//mu               sync.Mutex
}

func NewRepoClient(repoRootUrl string, logLevel string) *PipyRepoClient {
	return NewRepoClientWithTransport(
		repoRootUrl,
		&http.Transport{
			DisableKeepAlives:  false,
			MaxIdleConns:       10,
			IdleConnTimeout:    60 * time.Second,
			DisableCompression: false,
		},
		logLevel,
	)
}

func NewRepoClientWithTransport(repoRootUrl string, transport *http.Transport, logLevel string) *PipyRepoClient {
	return newRepoClientWithRepoRootUrlAndTransport(
		repoRootUrl,
		transport,
		logLevel,
	)
}

func newRepoClientWithRepoRootUrlAndTransport(repoRootUrl string, transport *http.Transport, logLevel string) *PipyRepoClient {
	repo := &PipyRepoClient{
		baseUrl:          repoRootUrl,
		defaultTransport: transport,
	}

	httpClient := resty.New().
		SetTransport(repo.defaultTransport).
		SetScheme("http").
		SetAllowGetMethodPayload(true).
		SetBaseURL(repo.baseUrl).
		SetTimeout(5 * time.Second)

	switch logLevel {
	case "debug", "trace":
		httpClient = httpClient.SetDebug(true).EnableTrace()
	}

	repo.httpClient = httpClient

	return repo
}

func (p *PipyRepoClient) codebaseExists(path string) (bool, *Codebase) {
	resp, err := p.httpClient.R().
		SetResult(&Codebase{}).
		Get(fullRepoApiPath(path))

	if err == nil {
		switch resp.StatusCode() {
		case http.StatusNotFound:
			return false, nil
		case http.StatusOK:
			return true, resp.Result().(*Codebase)
		}
	}

	log.Error().Msgf("error happened while getting path %q, %v", path, err)
	return false, nil
}

func (p *PipyRepoClient) get(path string) (*Codebase, error) {
	resp, err := p.httpClient.R().
		SetResult(&Codebase{}).
		Get(fullRepoApiPath(path))

	if err != nil {
		log.Error().Msgf("Failed to get path %q, error: %s", path, err.Error())
		return nil, err
	}

	return resp.Result().(*Codebase), nil
}

func (p *PipyRepoClient) createCodebase(path string) (*Codebase, error) {
	resp, err := p.httpClient.R().
		SetHeader("Content-Type", "application/json").
		SetBody(Codebase{Version: 1}).
		Post(fullRepoApiPath(path))

	if err != nil {
		log.Error().Msgf("failed to create codebase %q, error: %s", path, err.Error())
		return nil, err
	}

	if resp.IsError() {
		return nil, fmt.Errorf("failed to create codebase %q, reason: %s", path, resp.Status())
	}

	codebase, err := p.get(path)
	if err != nil {
		return nil, err
	}

	return codebase, nil
}

func (p *PipyRepoClient) deriveCodebase(path, base string) (*Codebase, error) {
	exists, _ := p.codebaseExists(base)
	if !exists {
		return nil, fmt.Errorf("parent %q of codebase %q doesn't exists", base, path)
	}

	resp, err := p.httpClient.R().
		SetHeader("Content-Type", "application/json").
		SetBody(Codebase{Version: 1, Base: base}).
		Post(fullRepoApiPath(path))

	if err != nil {
		log.Error().Msgf("Failed to derive codebase codebase: path: %q, base: %q, error: %s", path, base, err.Error())
		return nil, err
	}

	switch resp.StatusCode() {
	case http.StatusOK, http.StatusCreated:
		log.Debug().Msgf("Status code is %d, stands for success.", resp.StatusCode())
	default:
		log.Error().Msgf("Response contains error: %v", resp.Status())
		return nil, fmt.Errorf("failed to derive codebase codebase: path: %q, base: %q, reason: %s, %s", path, base, resp.Status(), resp.Body())
	}

	log.Debug().Msgf("Getting info of codebase %q", path)
	codebase, err := p.get(path)
	if err != nil {
		log.Debug().Msgf("Failed to get info of codebase %q", path)
		return nil, err
	}

	log.Debug().Msgf("Successfully derived codebase: %v", codebase)
	return codebase, nil
}

func (p *PipyRepoClient) GetFile(path string) (string, error) {
	resp, err := p.httpClient.R().
		Get(fullFileApiPath(path))

	if err != nil {
		log.Error().Msgf("Failed to get path %q, error: %s", path, err.Error())
		return "", err
	}

	result := string(resp.Body())
	log.Debug().Msgf("Content of %q:\n\n\n%s\n\n\n", path, result)

	return result, nil
}

func (p *PipyRepoClient) upsertFile(path string, content interface{}) error {
	// FIXME: temp solution, refine it later
	contentType := "text/plain"
	if strings.HasSuffix(path, ".json") {
		contentType = "application/json"
	}

	resp, err := p.httpClient.R().
		SetHeader("Content-Type", contentType).
		SetBody(content).
		Post(fullFileApiPath(path))

	if err != nil {
		log.Error().Msgf("error happened while trying to upsert %q to repo, %s", path, err.Error())
		return err
	}

	if resp.IsSuccess() {
		return nil
	}

	errstr := "repo server responsed with error HTTP code: %d, error: %s"
	log.Error().Msgf(errstr, resp.StatusCode(), resp.Status())
	return fmt.Errorf(errstr, resp.StatusCode(), resp.Status())
}

// deleteFile delete codebase file
func (p *PipyRepoClient) deleteFile(path string) (success bool, err error) {
	var resp *resty.Response

	resp, err = p.httpClient.R().
		Delete(fullFileApiPath(path))

	if err == nil {
		if resp.IsSuccess() {
			success = true
			return
		}
		err = fmt.Errorf("error happened while deleting codebase[%s], reason: %s", path, resp.Status())
		return
	}

	log.Err(err).Msgf("error happened while deleting codebase[%s]", path)
	return
}

// Commit the codebase, version is the current vesion of the codebase, it will be increased by 1 when committing
func (p *PipyRepoClient) commit(path string, _ int64) error {
	resp, err := p.httpClient.R().
		SetHeader("Content-Type", "application/json").
		SetBody(Codebase{Version: time.Now().UnixNano()}).
		SetResult(&Codebase{}).
		Patch(fullRepoApiPath(path))

	if err != nil {
		return err
	}

	if resp.IsSuccess() {
		return nil
	}

	err = fmt.Errorf("failed to commit codebase %q, reason: %s", path, resp.Status())
	log.Err(err)

	return err
}

// TODO: handle concurrent updating

func (p *PipyRepoClient) Batch(batches []Batch) error {
	if len(batches) == 0 {
		return nil
	}

	for _, batch := range batches {
		// 1. batch.Basepath, if not exists, create it
		log.Debug().Msgf("batch.Basepath = %q", batch.Basepath)
		var version int64
		exists, codebase := p.codebaseExists(batch.Basepath)
		if exists {
			// just get the version of codebase
			version = codebase.Version
		} else {
			log.Debug().Msgf("%q doesn't exist in repo", batch.Basepath)
			result, err := p.createCodebase(batch.Basepath)
			if err != nil {
				log.Error().Msgf("Not able to create the codebase %q, reason: %s", batch.Basepath, err.Error())
				return err
			}

			log.Debug().Msgf("Result = %v", result)

			version = result.Version
		}

		// 2. upload each json to repo
		for _, item := range batch.Items {
			fullpath := fmt.Sprintf("%s%s", batch.Basepath, item.String())
			log.Debug().Msgf("Creating/updating config %q", fullpath)
			log.Debug().Msgf("Content: %v", item.Content)
			err := p.upsertFile(fullpath, item.Content)
			if err != nil {
				log.Error().Msgf("Upsert %q error, reason: %s", fullpath, err.Error())
				return err
			}
		}

		for _, file := range batch.DelItems {
			fullpath := fmt.Sprintf("%s%s", batch.Basepath, file)
			log.Debug().Msgf("Deleting %q", fullpath)
			if _, err := p.deleteFile(fullpath); err != nil {
				return err
			}
		}

		// 3. commit the repo, so that changes can take effect
		log.Debug().Msgf("Committing batch.Basepath = %q", batch.Basepath)
		// NOT a valid version, ignore committing
		if version == -1 {
			err := fmt.Errorf("%d is not a valid version", version)
			log.Err(err)
			return err
		}
		if err := p.commit(batch.Basepath, version); err != nil {
			log.Error().Msgf("Error happened while committing the codebase %q, error: %s", batch.Basepath, err.Error())
			return err
		}
	}

	return nil
}

func (p *PipyRepoClient) DeriveCodebase(path, base string) error {
	log.Debug().Msgf("Checking if exists, codebase %q", path)
	exists, _ := p.codebaseExists(path)

	if exists {
		log.Debug().Msgf("Codebase %q already exists, ignore deriving ...", path)
	} else {
		log.Debug().Msgf("Codebase %q doesn't exist, deriving ...", path)
		result, err := p.deriveCodebase(path, base)
		if err != nil {
			log.Error().Msgf("Deriving codebase %q error: %v", path, err)
			return err
		}
		log.Debug().Msgf("Successfully derived codebase %q", path)

		log.Debug().Msgf("Committing the changes of codebase %q", path)
		if err = p.commit(path, result.Version); err != nil {
			log.Error().Msgf("Committing codebase %q error: %v", path, err)
			return err
		}
		log.Debug().Msgf("Successfully committed codebase %q", path)
	}

	return nil
}

func (p *PipyRepoClient) IsRepoUp() bool {
	_, err := p.get("/")

	return err == nil
}

func (p *PipyRepoClient) CodebaseExists(path string) bool {
	exists, _ := p.codebaseExists(path)

	return exists
}

func (p *PipyRepoClient) ListFiles(path string) ([]string, error) {
	codebase, err := p.get(path)

	if err != nil {
		return nil, err
	}

	return codebase.Files, nil
}

func fullRepoApiPath(path string) string {
	return fmt.Sprintf("%s%s", constants.DefaultPipyRepoAPIPath, path)
}

func fullFileApiPath(path string) string {
	return fmt.Sprintf("%s%s", constants.DefaultPipyFileAPIPath, path)
}
