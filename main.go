package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/youpy/go-wav"
)

type header struct {
	copyrightOffset      uint16
	encodingType         byte
	blockSize            byte
	sampleBitdepth       byte
	channelCount         byte
	sampleRate           uint32
	totalSamples         uint32
	highpassFrequency    uint16
	version              byte
	flags                byte
	loopAlignmentSamples uint16
	loopEnabled          bool
	loopBeginSampleIndex uint32
	loopBeginByteIndex   uint32
	loopEndSampleIndex   uint32
	loopEndByteIndex     uint32
}

func (h *header) Read(r *os.File) {
	buffer := make([]byte, 0x40)
	r.Seek(0, 0)
	r.Read(buffer)

	if !(buffer[0x0] == 0x80 && buffer[0x1] == 0x00) {
		panic("Magic does not match")
	}

	h.copyrightOffset = binary.BigEndian.Uint16(buffer[0x02:0x04])
	h.encodingType = buffer[0x04]
	h.blockSize = buffer[0x05]
	h.sampleBitdepth = buffer[0x06]
	h.channelCount = buffer[0x07]
	h.sampleRate = binary.BigEndian.Uint32(buffer[0x08:0x0C])
	h.totalSamples = binary.BigEndian.Uint32(buffer[0x0C:0x10])
	h.highpassFrequency = binary.BigEndian.Uint16(buffer[0x10:0x12])
	h.version = buffer[0x12]
	h.flags = buffer[0x13]

	switch h.version {
	case 3:
		h.loopAlignmentSamples = binary.BigEndian.Uint16(buffer[0x14:0x16])
		h.loopEnabled = binary.BigEndian.Uint32(buffer[0x18:0x1C]) == 1
		h.loopBeginSampleIndex = binary.BigEndian.Uint32(buffer[0x1C:0x20])
		h.loopBeginByteIndex = binary.BigEndian.Uint32(buffer[0x20:0x24])
		h.loopEndSampleIndex = binary.BigEndian.Uint32(buffer[0x24:0x28])
		h.loopEndByteIndex = binary.BigEndian.Uint32(buffer[0x28:0x2C])
	case 4:
		h.loopEnabled = binary.BigEndian.Uint32(buffer[0x24:0x28]) == 1
		h.loopBeginSampleIndex = binary.BigEndian.Uint32(buffer[0x28:0x2C])
		h.loopBeginByteIndex = binary.BigEndian.Uint32(buffer[0x2C:0x30])
		h.loopEndSampleIndex = binary.BigEndian.Uint32(buffer[0x30:0x34])
		h.loopEndByteIndex = binary.BigEndian.Uint32(buffer[0x34:0x38])
	}
}

func (h *header) Write(w *os.File) {
	buffer := make([]byte, 0x40)
	buffer[0x0] = 0x80
	buffer[0x1] = 0x00

	binary.BigEndian.PutUint16(buffer[0x02:0x04], h.copyrightOffset)
	buffer[0x04] = h.encodingType
	buffer[0x05] = h.blockSize
	buffer[0x06] = h.sampleBitdepth
	buffer[0x07] = h.channelCount
	binary.BigEndian.PutUint32(buffer[0x08:0x0C], h.sampleRate)
	binary.BigEndian.PutUint32(buffer[0x0C:0x10], h.totalSamples)
	binary.BigEndian.PutUint16(buffer[0x10:0x12], h.highpassFrequency)
	buffer[0x12] = h.version
	buffer[0x13] = h.flags

	switch h.version {
	case 3:
		binary.BigEndian.PutUint16(buffer[0x14:0x16], h.loopAlignmentSamples)
		if h.loopEnabled {
			binary.BigEndian.PutUint32(buffer[0x18:0x1C], uint32(1))
		}
		binary.BigEndian.PutUint32(buffer[0x1C:0x20], h.loopBeginSampleIndex)
		binary.BigEndian.PutUint32(buffer[0x20:0x24], h.loopBeginByteIndex)
		binary.BigEndian.PutUint32(buffer[0x24:0x28], h.loopEndSampleIndex)
		binary.BigEndian.PutUint32(buffer[0x28:0x2C], h.loopEndByteIndex)
	case 4:
		if h.loopEnabled {
			binary.BigEndian.PutUint32(buffer[0x24:0x28], uint32(1))
		}
		binary.BigEndian.PutUint32(buffer[0x28:0x2C], h.loopBeginSampleIndex)
		binary.BigEndian.PutUint32(buffer[0x2C:0x30], h.loopBeginByteIndex)
		binary.BigEndian.PutUint32(buffer[0x30:0x34], h.loopEndSampleIndex)
		binary.BigEndian.PutUint32(buffer[0x34:0x38], h.loopEndByteIndex)
	}

	w.Seek(0, 0)
	w.Write(buffer)
}

func (h *header) SetLoopBytes(samplesPerBlock byte) {
	beginFrame := h.loopBeginSampleIndex / uint32(samplesPerBlock) * uint32(h.blockSize) * uint32(h.channelCount)
	h.loopBeginByteIndex = uint32(h.copyrightOffset) + 4 + beginFrame
	endFrame := (h.loopEndSampleIndex/uint32(samplesPerBlock) + 1) * uint32(h.blockSize) * uint32(h.channelCount)
	h.loopEndByteIndex = uint32(h.copyrightOffset) + 4 + endFrame
}

func main() {
	if len(os.Args) != 3 {
		fmt.Println("用法: adxtool -e <输入.wav>")
		fmt.Println("       adxtool -d <输入.adx>")
		os.Exit(1)
	}

	mode := os.Args[1]
	inputFile := os.Args[2]

	switch mode {
	case "-e":
		if !strings.HasSuffix(strings.ToLower(inputFile), ".wav") {
			fmt.Println("错误: 输入文件必须是 .wav 格式")
			os.Exit(1)
		}
		wav2adx(inputFile)
	case "-d":
		if !strings.HasSuffix(strings.ToLower(inputFile), ".adx") {
			fmt.Println("错误: 输入文件必须是 .adx 格式")
			os.Exit(1)
		}
		adx2wav(inputFile)
	default:
		fmt.Println("用法: adxtool -e <输入.wav>")
		fmt.Println("       adxtool -d <输入.adx>")
		os.Exit(1)
	}
}

func adx2wav(inPath string) {
	startTime := time.Now()
	outPath := strings.TrimSuffix(inPath, ".adx") + ".wav"

	outFile, err := os.Create(outPath)
	if err != nil {
		panic(err)
	}
	defer outFile.Close()

	inFile, err := os.Open(inPath)
	if err != nil {
		panic(err)
	}
	defer inFile.Close()

	adx := header{}
	adx.Read(inFile)

	writer := wav.NewWriter(outFile, uint32(adx.totalSamples), uint16(adx.channelCount), uint32(adx.sampleRate), 16)

	a := math.Sqrt(2) - math.Cos(2*math.Pi*float64(adx.highpassFrequency)/float64(adx.sampleRate))
	b := math.Sqrt(2) - 1
	c := (a - math.Sqrt((a+b)*(a-b))) / b

	coefficient := make([]float64, 2)
	coefficient[0] = c * 2
	coefficient[1] = -(c * c)

	pastSamples := make([]int32, 2*adx.channelCount)
	sampleIndex := uint32(0)

	samplesPerBlock := (adx.blockSize - 2) * 8 / adx.sampleBitdepth
	scale := make([]uint16, adx.channelCount)

	for sampleIndex < adx.totalSamples {
		start := uint32(adx.copyrightOffset) + 4 + sampleIndex/uint32(samplesPerBlock)*uint32(adx.blockSize)*uint32(adx.channelCount)

		buffer := make([]byte, uint32(adx.blockSize)*uint32(adx.channelCount))
		inFile.Seek(int64(start), 0)
		inFile.Read(buffer)

		outBuffer := make([]wav.Sample, samplesPerBlock)

		for i := byte(0); i < adx.channelCount; i++ {
			scaleBytes := make([]byte, 2)
			inFile.Seek(int64(start+uint32(adx.blockSize)*uint32(i)), 0)
			inFile.Read(scaleBytes)
			scale[i] = binary.BigEndian.Uint16(scaleBytes)
		}

		for sampleOffset := 0; sampleOffset < int(samplesPerBlock); sampleOffset += 2 {
			outSamples := make([]int, 2*adx.channelCount)

			for i := byte(0); i < adx.channelCount; i++ {
				sampleErrorNibbles := make([]byte, 2)
				sampleErrorNibbles[0] = buffer[uint32(adx.blockSize)*uint32(i)+2+uint32(sampleOffset)/2] >> 4
				sampleErrorNibbles[1] = buffer[uint32(adx.blockSize)*uint32(i)+2+uint32(sampleOffset)/2] & 0xF

				for nibbleIdx, v := range sampleErrorNibbles {
					samplePrediction := coefficient[0]*float64(pastSamples[i*2+0]) + coefficient[1]*float64(pastSamples[i*2+1])
					sampleError := int32(v)
					sampleError = (sampleError << 28) >> 28
					sampleError *= int32(scale[i])
					sample := sampleError + int32(samplePrediction)

					pastSamples[i*2+1] = pastSamples[i*2+0]
					pastSamples[i*2+0] = sample

					if sample > 32767 {
						sample = 32767
					} else if sample < -32768 {
						sample = -32768
					}
					outSamples[int(i)*2+nibbleIdx] = int(sample)
				}
			}

			switch adx.channelCount {
			case 1:
				outBuffer[sampleOffset+0] = wav.Sample{[2]int{outSamples[0], 0}}
				outBuffer[sampleOffset+1] = wav.Sample{[2]int{outSamples[1], 0}}
			case 2:
				outBuffer[sampleOffset+0] = wav.Sample{[2]int{outSamples[0], outSamples[2]}}
				outBuffer[sampleOffset+1] = wav.Sample{[2]int{outSamples[1], outSamples[3]}}
			}
			sampleIndex += 2
		}
		writer.WriteSamples(outBuffer)
	}
	fmt.Printf("转换完成: %s -> %s (%.2f秒)\n", inPath, outPath, time.Since(startTime).Seconds())
}

func wav2adx(inPath string) {
	startTime := time.Now()
	outPath := strings.TrimSuffix(inPath, ".wav") + ".adx"

	outFile, err := os.Create(outPath)
	if err != nil {
		panic(err)
	}
	defer outFile.Close()

	inFile, err := os.Open(inPath)
	if err != nil {
		panic(err)
	}
	defer inFile.Close()

	reader := wav.NewReader(inFile)
	format, err := reader.Format()
	if err != nil {
		panic(err)
	}

	adx := header{
		copyrightOffset:   404,
		encodingType:      0x03,
		blockSize:         18,
		sampleBitdepth:    4,
		channelCount:      byte(format.NumChannels),
		sampleRate:        format.SampleRate,
		highpassFrequency: 2000,
		version:           4,
		flags:             0,
		loopEnabled:       false,
	}

	a := math.Sqrt(2) - math.Cos(2*math.Pi*float64(adx.highpassFrequency)/float64(adx.sampleRate))
	b := math.Sqrt(2) - 1
	c := (a - math.Sqrt((a+b)*(a-b))) / b

	coefficient := make([]float64, 2)
	coefficient[0] = c * 2
	coefficient[1] = -(c * c)

	pastSamples := make([]int32, 2*adx.channelCount)
	sampleIndex := uint32(0)
	samplesPerBlock := (adx.blockSize - 2) * 8 / adx.sampleBitdepth
	adx.SetLoopBytes(samplesPerBlock)

	for {
		buffer, err := reader.ReadSamples(uint32(samplesPerBlock))
		if err != nil {
			break
		}

		start := uint32(adx.copyrightOffset) + 4 + sampleIndex/uint32(samplesPerBlock)*uint32(adx.blockSize)*uint32(adx.channelCount)
		scaledSampleErrorNibbles := make([]int32, uint32(adx.channelCount)*uint32(samplesPerBlock))

		samplesCanGet := samplesPerBlock
		if byte(len(buffer)) < samplesPerBlock {
			samplesCanGet = byte(len(buffer))
		}

		for sampleOffset := byte(0); sampleOffset < samplesCanGet; sampleOffset++ {
			inSamples := buffer[sampleOffset].Values

			for i := byte(0); i < adx.channelCount; i++ {
				samplePrediction := coefficient[0]*float64(pastSamples[i*2+0]) + coefficient[1]*float64(pastSamples[i*2+1])
				sample := int32(inSamples[i])
				scaledSampleErrorNibbles[samplesPerBlock*byte(i)+sampleOffset] = sample - int32(samplePrediction)

				pastSamples[i*2+1] = pastSamples[i*2+0]
				pastSamples[i*2+0] = sample
			}
			sampleIndex++
		}

		scale := generateScale(&adx, samplesPerBlock, scaledSampleErrorNibbles)
		sampleErrorBytes := generateSampleError(&adx, samplesPerBlock, scaledSampleErrorNibbles, scale)

		for i := byte(0); i < adx.channelCount; i++ {
			scaleBytes := make([]byte, 2)
			binary.BigEndian.PutUint16(scaleBytes, scale[i])
			outFile.Seek(int64(start+uint32(adx.blockSize)*uint32(i)), 0)
			outFile.Write(scaleBytes)

			sectionLen := len(sampleErrorBytes) / int(adx.channelCount)
			outFile.Seek(int64(start+2+uint32(adx.blockSize)*uint32(i)), 0)
			outFile.Write(sampleErrorBytes[sectionLen*int(i) : sectionLen*int(i+1)])
		}
	}

	adx.totalSamples = sampleIndex
	adx.Write(outFile)
	outFile.Seek(int64(adx.copyrightOffset-2), 0)
	outFile.Write([]byte("(c)CRI"))

	fmt.Printf("转换完成: %s -> %s (%.2f秒)\n", inPath, outPath, time.Since(startTime).Seconds())
}

func generateScale(adx *header, samplesPerBlock byte, scaledSampleErrorNibbles []int32) []uint16 {
	scale := make([]uint16, adx.channelCount)

	for i := byte(0); i < adx.channelCount; i++ {
		minAbsErr := scaledSampleErrorNibbles[samplesPerBlock*i+0]
		maxAbsErr := scaledSampleErrorNibbles[samplesPerBlock*i+0]

		for j := byte(0); j < samplesPerBlock; j++ {
			v := scaledSampleErrorNibbles[samplesPerBlock*i+j]
			if v > maxAbsErr {
				maxAbsErr = v
			}
			if v < minAbsErr {
				minAbsErr = v
			}
		}

		if maxAbsErr > 0 && minAbsErr < 0 {
			if maxAbsErr > -minAbsErr {
				scale[i] = uint16(maxAbsErr / 7)
			} else {
				scale[i] = uint16(minAbsErr / -8)
			}
		} else if minAbsErr > 0 {
			scale[i] = uint16(maxAbsErr / 7)
		} else if maxAbsErr < 0 {
			scale[i] = uint16(minAbsErr / -8)
		}
	}
	return scale
}

func generateSampleError(adx *header, samplesPerBlock byte, scaledSampleErrorNibbles []int32, scale []uint16) []byte {
	sampleErrorNibbles := make([]byte, len(scaledSampleErrorNibbles))

	for i := byte(0); i < adx.channelCount; i++ {
		for j := byte(0); j < samplesPerBlock; j++ {
			unscaledError := byte(0)
			if scale[i] != 0 {
				unscaledError = byte(scaledSampleErrorNibbles[samplesPerBlock*i+j] / int32(scale[i]))
			}
			sampleErrorNibbles[samplesPerBlock*i+j] = unscaledError
		}
	}

	sampleErrorBytes := make([]byte, len(sampleErrorNibbles)/2)
	for i := 0; i < len(sampleErrorNibbles); i += 2 {
		sampleErrorBytes[i/2] = (sampleErrorNibbles[i+0] << 4) | (sampleErrorNibbles[i+1] & 0xF)
	}
	return sampleErrorBytes
}