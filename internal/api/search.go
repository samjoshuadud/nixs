package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	baseURL  = "https://search.nixos.org/backend"
	// Public credentials used by search.nixos.org frontend
	authUser = "aWVSALXpZv"
	authPass = "X8gPHnzL52wFEekuxsfQ9cSh"
	// Schema version — NixOS bumps this periodically
	esSchema = "48"
)

// Package represents a nixpkgs package result
type Package struct {
	Name        string   `json:"package_attr_name"`
	Version     string   `json:"package_pversion"`
	Description string   `json:"package_description"`
	LongDesc    string   `json:"package_longDescription"`
	Programs    []string `json:"package_programs"`
	Homepage    []string `json:"package_homepage"`
	License     []struct {
		FullName string `json:"fullName"`
	} `json:"package_license"`
	Maintainers []struct {
		Name string `json:"name"`
	} `json:"package_maintainers"`
}

// Option represents a NixOS or Home Manager option result
type Option struct {
	Name        string `json:"option_name"`
	Description string `json:"option_description"`
	Type        string `json:"option_type"`
	Default     string `json:"option_default"`
	Example     string `json:"option_example"`
}

type esResponse struct {
	Hits struct {
		Hits []struct {
			Source json.RawMessage `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

func SearchPackages(query, channel string, max int) ([]Package, error) {
	body := map[string]any{
		"from": 0,
		"size": max,
		"query": map[string]any{
			"multi_match": map[string]any{
				"query": query,
				"fields": []string{
					"package_attr_name^9",
					"package_attr_name_reverse^2",
					"package_programs^9",
					"package_description^1.3",
					"package_longDescription^1",
				},
				"type": "cross_fields",
			},
		},
	}

	url := fmt.Sprintf("%s/latest-%s-nixos-%s/_search", baseURL, esSchema, channel)
	raw, err := doRequest(url, body)
	if err != nil {
		return nil, err
	}

	var results []Package
	for _, hit := range raw.Hits.Hits {
		var p Package
		if err := json.Unmarshal(hit.Source, &p); err != nil {
			continue
		}
		results = append(results, p)
	}
	return results, nil
}

func SearchOptions(query, channel string, max int) ([]Option, error) {
	optChannel := fmt.Sprintf("latest-%s-nixos-%s-options", esSchema, channel)
	body := map[string]any{
		"from": 0,
		"size": max,
		"query": map[string]any{
			"multi_match": map[string]any{
				"query":  query,
				"fields": []string{"option_name^6", "option_description^1"},
				"type":   "cross_fields",
			},
		},
	}

	url := fmt.Sprintf("%s/%s/_search", baseURL, optChannel)
	return doOptionRequest(url, body)
}

func SearchHomeManager(query string, max int) ([]Option, error) {
	body := map[string]any{
		"from": 0,
		"size": max,
		"query": map[string]any{
			"multi_match": map[string]any{
				"query":  query,
				"fields": []string{"option_name^6", "option_description^1"},
				"type":   "cross_fields",
			},
		},
	}

	// Home Manager uses its own index
	url := fmt.Sprintf("%s/latest-%s-home-manager/_search", baseURL, esSchema)
	return doOptionRequest(url, body)
}

func doRequest(url string, body any) (*esResponse, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(authUser, authPass)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned %d", resp.StatusCode)
	}

	var result esResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &result, nil
}

func doOptionRequest(url string, body any) ([]Option, error) {
	raw, err := doRequest(url, body)
	if err != nil {
		return nil, err
	}

	var results []Option
	for _, hit := range raw.Hits.Hits {
		var o Option
		if err := json.Unmarshal(hit.Source, &o); err != nil {
			continue
		}
		results = append(results, o)
	}
	return results, nil
}
