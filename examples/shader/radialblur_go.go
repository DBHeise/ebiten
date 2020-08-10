// Code generated by file2byteslice. DO NOT EDIT.
// (gofmt is fine after generating)

package main

var radialblur_go = []byte("// Copyright 2020 The Ebiten Authors\n//\n// Licensed under the Apache License, Version 2.0 (the \"License\");\n// you may not use this file except in compliance with the License.\n// You may obtain a copy of the License at\n//\n//     http://www.apache.org/licenses/LICENSE-2.0\n//\n// Unless required by applicable law or agreed to in writing, software\n// distributed under the License is distributed on an \"AS IS\" BASIS,\n// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.\n// See the License for the specific language governing permissions and\n// limitations under the License.\n\n// +build ignore\n\npackage main\n\nvar Time float\nvar Cursor vec2\nvar ScreenSize vec2\n\nfunc Fragment(position vec4, texCoord vec2, color vec4) vec4 {\n\tdir := normalize(position.xy - Cursor)\n\tclr := image2TextureAt(texCoord)\n\n\tsamples := [10]float{\n\t\t-22, -14, -8, -4, -2, 2, 4, 8, 14, 22,\n\t}\n\t// TODO: Add len(samples)\n\tsum := clr\n\tfor i := 0; i < 10; i++ {\n\t\tpos := texCoord + dir*samples[i]/image2TextureSize()\n\t\tsum += image2TextureBoundsAt(pos)\n\t}\n\tsum /= 10 + 1\n\n\tdist := distance(position.xy, Cursor)\n\tt := clamp(dist/256, 0, 1)\n\treturn mix(clr, sum, t)\n}\n")
