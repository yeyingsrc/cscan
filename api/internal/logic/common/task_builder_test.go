package common

import (
	"testing"

	"cscan/model"
	"cscan/scanner"
)

func TestCollectInitialAssetsWithGenerator_DeduplicatesByAuthorityAndHostPort(t *testing.T) {
	generator := func(batch string) []*scanner.Asset {
		switch batch {
		case "batch1":
			return []*scanner.Asset{
				{Host: "example.com", Authority: "example.com", Category: "domain", IPV4: []scanner.IPInfo{{IP: "1.2.3.4"}}},
				{Host: "example.com", Authority: "example.com", Category: "domain", IPV4: []scanner.IPInfo{{IP: "1.2.3.4"}}},
				{Host: "example.com", Authority: "example.com", Port: 443, Category: "domain", Service: "https", IPV4: []scanner.IPInfo{{IP: "1.2.3.4"}}},
			}
		case "batch2":
			return []*scanner.Asset{
				{Host: "example.com", Authority: "example.com", Category: "domain", IPV4: []scanner.IPInfo{{IP: "5.6.7.8"}}},
				{Host: "example.com", Authority: "example.com", Port: 443, Category: "domain", Service: "https", IPV4: []scanner.IPInfo{{IP: "5.6.7.8"}}},
				{Host: "1.1.1.1", Authority: "1.1.1.1", Category: "ipv4"},
			}
		default:
			return nil
		}
	}

	assets := collectInitialAssetsWithGenerator([]string{"batch1", "batch2"}, generator)
	if len(assets) != 3 {
		t.Fatalf("expected 3 unique assets, got %d", len(assets))
	}

	if assets[0].Authority != "example.com" || assets[0].Port != 0 {
		t.Fatalf("expected first asset to be bare domain, got authority=%s port=%d", assets[0].Authority, assets[0].Port)
	}
	if assets[1].Authority != "example.com" || assets[1].Port != 443 {
		t.Fatalf("expected second asset to be host:443, got authority=%s port=%d", assets[1].Authority, assets[1].Port)
	}
	if assets[2].Host != "1.1.1.1" || assets[2].Category != "ipv4" {
		t.Fatalf("expected third asset to be raw ip target, got host=%s category=%s", assets[2].Host, assets[2].Category)
	}
}

func TestConvertScannerAssetToModelAsset_FillsMinimalFields(t *testing.T) {
	scanAsset := &scanner.Asset{
		Authority: "example.com",
		Host:      "example.com",
		Category:  "domain",
		IsHTTP:    true,
		CName:     "alias.example.com",
		IPV4:      []scanner.IPInfo{{IP: "1.2.3.4"}},
		IPV6:      []scanner.IPInfo{{IP: "2001:db8::1"}},
	}

	asset := convertScannerAssetToModelAsset(&model.MainTask{TaskId: "t1"}, scanAsset, "org-1")
	if asset.Authority != "example.com" || asset.Domain != "example.com" {
		t.Fatalf("expected authority/domain to be example.com, got authority=%s domain=%s", asset.Authority, asset.Domain)
	}
	if asset.TaskId != "t1" || asset.OrgId != "org-1" {
		t.Fatalf("expected task/org ids to be preserved, got taskId=%s orgId=%s", asset.TaskId, asset.OrgId)
	}
	if asset.Source != "user_input" {
		t.Fatalf("expected default source user_input, got %s", asset.Source)
	}
	if len(asset.Ip.IpV4) != 1 || asset.Ip.IpV4[0].IPName != "1.2.3.4" {
		t.Fatalf("expected ipv4 to be converted, got %+v", asset.Ip.IpV4)
	}
	if len(asset.Ip.IpV6) != 1 || asset.Ip.IpV6[0].IPName != "2001:db8::1" {
		t.Fatalf("expected ipv6 to be converted, got %+v", asset.Ip.IpV6)
	}
	if asset.CName != "alias.example.com" || !asset.IsHTTP {
		t.Fatalf("expected cname and isHttp to be preserved, got cname=%s isHttp=%v", asset.CName, asset.IsHTTP)
	}
}

func TestBuildPrewriteAssetKey_UsesAuthorityOrHostPort(t *testing.T) {
	if key := buildPrewriteAssetKey(&scanner.Asset{Host: "example.com", Authority: "example.com"}); key != "example.com" {
		t.Fatalf("expected bare domain key example.com, got %s", key)
	}
	if key := buildPrewriteAssetKey(&scanner.Asset{Host: "example.com", Authority: "example.com", Port: 443}); key != "example.com:443" {
		t.Fatalf("expected host:port key example.com:443, got %s", key)
	}
}

func TestCollectInitialAssetsWithGenerator_SkipsEmptyHost(t *testing.T) {
	generator := func(batch string) []*scanner.Asset {
		return []*scanner.Asset{{Authority: "missing-host"}, nil, {Host: "ok", Authority: "ok"}}
	}
	assets := collectInitialAssetsWithGenerator([]string{"batch"}, generator)
	if len(assets) != 1 || assets[0].Host != "ok" {
		t.Fatalf("expected only valid host asset, got %+v", assets)
	}
}
