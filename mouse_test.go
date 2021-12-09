package rdesktop

import (
	"image/jpeg"
	"os"
	"testing"
)

func TestMouseMove(t *testing.T) {
	cli, err := New()
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()
	cli.ShowCursor(true)
	err = cli.MouseMove(100, 100)
	if err != nil {
		t.Fatal(err)
	}
	img, err := cli.Screenshot()
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Create("screenshot.jpg")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	err = jpeg.Encode(f, img, nil)
	if err != nil {
		t.Fatal(err)
	}
}
