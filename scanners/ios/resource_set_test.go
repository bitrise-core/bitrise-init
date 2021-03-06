package ios

import (
	"bytes"
	"io"
	"reflect"
	"testing"
)

func Test_parseAppIconMetadata(t *testing.T) {
	tests := []struct {
		name    string
		input   io.Reader
		want    []appIcon
		wantErr bool
	}{
		{
			name:  "Minimal",
			input: bytes.NewReader(jsonRaw),
			want: []appIcon{
				{
					Size:     20,
					Filename: "Icon-App-20x20@2x.png",
				},
				{
					Size:     1024,
					Filename: "ItunesArtwork@2x.png",
				},
			},
			wantErr: false,
		},
		{
			name:  "Full",
			input: bytes.NewReader(jsonRawMissingSize),
			want: []appIcon{
				{
					Size:     20,
					Filename: "Icon-App-20x20@2x.png",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseResourceSetMetadata(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseAppIconMetadata() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseAppIconMetadata() = %v, want %v", got, tt.want)
			}
		})
	}
}

var jsonRaw = []byte(`
{
	"images" : [
		{
		"size" : "20x20",
		"idiom" : "iphone",
		"filename" : "Icon-App-20x20@2x.png",
		"scale" : "2x"
		},
		{
		"size" : "1024x1024",
		"idiom" : "ios-marketing",
		"filename" : "ItunesArtwork@2x.png",
		"scale" : "1x"
		}
	],
	"info" : {
		"version" : 1,
		"author" : "xcode"
	}
	}
`)

var jsonRawMissingSize = []byte(`
{
	"images" : [
		{
		"size" : "20x20",
		"idiom" : "iphone",
		"filename" : "Icon-App-20x20@2x.png",
		"scale" : "2x"
		},
		{
			"size" : "1024x1024",
			"idiom" : "ios-marketing",
			"scale" : "1x"
		}
	],
	"info" : {
		"version" : 1,
		"author" : "xcode"
	}
	}
`)
