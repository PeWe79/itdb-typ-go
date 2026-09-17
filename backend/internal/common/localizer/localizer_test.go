package localizer

import "testing"

func TestLocalizeMessage(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"invoice file extension", "invoice file must be an image or pdf", "发票类型的文件仅支持图片或 PDF 格式"},
		{"floorplan extension", "floorplan file must be an image", "建筑平面图仅支持图片文件扩展名"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := LocalizeMessage(tc.in); got != tc.want {
				t.Fatalf("LocalizeMessage(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
