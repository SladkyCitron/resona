# 🔊 Resona

[![Go Reference](https://pkg.go.dev/badge/github.com/SladkyCitron/resona.svg)](https://pkg.go.dev/github.com/SladkyCitron/resona) [![CI (Go)](https://github.com/SladkyCitron/resona/actions/workflows/ci.yml/badge.svg)](https://github.com/SladkyCitron/resona/actions/workflows/ci.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/SladkyCitron/resona)](https://goreportcard.com/report/github.com/SladkyCitron/resona) [![GitHub license](https://img.shields.io/github/license/SladkyCitron/resona)](LICENSE) [![Made in Slovakia](https://raw.githubusercontent.com/pedromxavier/flag-badges/refs/heads/main/badges/SK.svg)](https://www.youtube.com/watch?v=UqXJ0ktrmh0)

**Resona** is the friendly, fast, and flexible audio and DSP toolkit for Go.
Whether you're building a synth, sequencer, effect, player, or a full-on singing voice synthesizer, Resona provides the tools you need, without getting in your way.

_Resona_ comes from the Latin word "resono", meaning "to resonate".

## ✨ Features

- Super lightweight: no bloat, just clean Go code (with a few optional deps)
- Modular, Go stdlib-style API for audio/DSP
- Supports loading WAV, MP3, FLAC, and much more!
- Core DSP math: windows, filters, oscillators, etc...
- Basic audio effects: gain, filter, etc...
- Basic generators: noise, oscillator, etc...

## 🚀 Getting Started

Install using this command:

```bash
go get github.com/SladkyCitron/resona
```

## 🧩 Conventions & Audio Data Model

Resona represents all audio data **interleaved 32-bit float samples** in the range [-1.0, 1.0].
For more details, see the [documentation](https://pkg.go.dev/github.com/SladkyCitron/resona).

## 📚 Documentation

All documentation is available at [pkg.go.dev/github.com/SladkyCitron/resona](https://pkg.go.dev/github.com/SladkyCitron/resona).

## ⚖️ License

Copyright © 2025 SladkyCitron

Licensed under the **MIT License** (see [LICENSE](LICENSE)) - free to use, fork, remix, and share!
