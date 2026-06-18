package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

const (
	baseURL  = "https://search.nixos.org/backend"
	// Public credentials used by search.nixos.org frontend
	authUser = "aWVSALXpZv"
	authPass = "X8gPHnzL52wFEekuxsfQ9cSh"

	hmBaseURL = "https://home-manager-options.extranix.com/data"
)

// Schema version — NixOS bumps this periodically
var esSchema = "48"

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

type hmOption struct {
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Type        string          `json:"type"`
	Default     json.RawMessage `json:"default"`
	Example     json.RawMessage `json:"example"`
}

func rawToString(raw json.RawMessage) string {
	if raw == nil || string(raw) == "null" {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	return string(raw)
}

type hmResponse struct {
	Options []hmOption `json:"options"`
}

func SearchPackages(query, channel string, max int) ([]Package, error) {
	body := map[string]any{
		"from": 0,
		"size": max,
		"query": map[string]any{
			"bool": map[string]any{
				"filter": []map[string]any{
					{"term": map[string]any{"type": map[string]any{"value": "package"}}},
				},
				"must": []map[string]any{
					{
						"dis_max": map[string]any{
							"tie_breaker": 0.7,
							"queries": []map[string]any{
								{
									"multi_match": map[string]any{
										"type":     "cross_fields",
										"query":    query,
										"analyzer": "whitespace",
										"operator": "and",
										"fields": []string{
											"package_attr_name^9",
											"package_attr_name.*^5.4",
											"package_pname^6",
											"package_pname.*^3.6",
											"package_description^1.3",
											"package_longDescription^1",
										},
									},
								},
								{
									"multi_match": map[string]any{
										"type":      "best_fields",
										"query":     query,
										"analyzer":  "whitespace",
										"operator":  "and",
										"fields":    []string{"package_programs^7.5"},
										"fuzziness": 1,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	raw, err := executeESRequest(body, channel, false)
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
	body := map[string]any{
		"from": 0,
		"size": max,
		"query": map[string]any{
			"bool": map[string]any{
				"must": []any{
					map[string]any{
						"multi_match": map[string]any{
							"query":  query,
							"fields": []string{"option_name^6", "option_description^1"},
							"type":   "cross_fields",
						},
					},
				},
				"filter": []any{
					map[string]any{
						"term": map[string]any{
							"type": "option",
						},
					},
				},
			},
		},
	}

	raw, err := executeESRequest(body, channel, false)
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

func SearchHomeManager(query string, max int) ([]Option, error) {
	url := fmt.Sprintf("%s/options-master.json", hmBaseURL)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Home Manager options: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Home Manager options API returned %d", resp.StatusCode)
	}

	var data hmResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode Home Manager options: %w", err)
	}

	queryLower := strings.ToLower(query)
	var results []Option
	for _, o := range data.Options {
		nameLower := strings.ToLower(o.Title)
		descLower := strings.ToLower(o.Description)
		if strings.Contains(nameLower, queryLower) || strings.Contains(descLower, queryLower) {
			results = append(results, Option{
				Name:        o.Title,
				Description: o.Description,
				Type:        o.Type,
				Default:     rawToString(o.Default),
				Example:     rawToString(o.Example),
			})
			if len(results) >= max {
				break
			}
		}
	}
	return results, nil
}

func executeESRequest(body any, channel string, retried bool) (*esResponse, error) {
	url := fmt.Sprintf("%s/latest-%s-nixos-%s/_search", baseURL, esSchema, channel)

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

	if resp.StatusCode == http.StatusNotFound && !retried {
		if newSchema, ok := resolveSchema(channel); ok {
			esSchema = newSchema
			return executeESRequest(body, channel, true)
		}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned %d", resp.StatusCode)
	}

	var result esResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &result, nil
}

func resolveSchema(channel string) (string, bool) {
	cur, err := strconv.Atoi(esSchema)
	if err != nil {
		return "", false
	}
	for k := 5; k >= 1; k-- {
		cand := strconv.Itoa(cur + k)
		if probeSchema(cand, channel) {
			return cand, true
		}
	}
	for k := 1; k <= 5; k++ {
		n := cur - k
		if n < 1 {
			break
		}
		cand := strconv.Itoa(n)
		if probeSchema(cand, channel) {
			return cand, true
		}
	}
	return "", false
}

func probeSchema(schema, channel string) bool {
	url := fmt.Sprintf("%s/latest-%s-nixos-%s", baseURL, schema, channel)
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return false
	}
	req.SetBasicAuth(authUser, authPass)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
