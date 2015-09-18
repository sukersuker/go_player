package go_player

import (
	"github.com/3d0c/gmf"
	player_log "github.com/flexconstructor/go_player/log"
	"sync"
)
/*
	Encode frames to jpeg images.
 */
type FFmpegEncoder struct {
	srcCodec     *gmf.CodecCtx        // video codec
	broadcast    chan []byte          // channel for result images
	error        chan *WSError        // error channel
	log          player_log.Logger    // logger
	close_chan   chan bool            // channel for closing encoder
	frame_cannel chan *gmf.Frame      // channel of source frames.
	wg           *sync.WaitGroup      // wait group for closing all encode goroutines.
}

// Run encoder.
func (e *FFmpegEncoder) Run() {
	defer e.Close()
	// get codec for jpeg encode.
	codec, err := gmf.FindEncoder(gmf.AV_CODEC_ID_MJPEG)
	if err != nil {
		e.error <- NewError(2, 1)
		return
	}
	e.log.Debug("encoder run")
	cc := gmf.NewCodecCtx(codec)
	defer gmf.Release(cc)
	// setts the properties of encode codec
	cc.SetPixFmt(gmf.AV_PIX_FMT_YUVJ420P)
	cc.SetWidth(e.srcCodec.Width())
	cc.SetHeight(e.srcCodec.Height())
	cc.SetTimeBase(e.srcCodec.TimeBase().AVR())

	if codec.IsExperimental() {
		cc.SetStrictCompliance(gmf.FF_COMPLIANCE_EXPERIMENTAL)
	}

	if err := cc.Open(nil); err != nil {
		e.log.Error("can not open codec")
		e.error <- NewError(3, 1)
		return
	}

	swsCtx := gmf.NewSwsCtx(e.srcCodec, cc, gmf.SWS_BICUBIC)
	defer gmf.Release(swsCtx)

	// convert to RGB, optionally resize could be here
	dstFrame := gmf.NewFrame().
		SetWidth(e.srcCodec.Width()).
		SetHeight(e.srcCodec.Height()).
		SetFormat(gmf.AV_PIX_FMT_YUVJ420P)
	defer gmf.Release(dstFrame)

	if err := dstFrame.ImgAlloc(); err != nil {
		e.log.Error("codec error: ", err)
		e.error <- NewError(4, 2)
		return
	}

	for {

		srcFrame, ok := <-e.frame_cannel
		if !ok {
			e.log.Error("frame is invalid")
			return
		}

		swsCtx.Scale(srcFrame, dstFrame)

		if p, ready, _ := dstFrame.EncodeNewPacket(cc); ready {
			e.broadcast <- p.Data()
			e.log.Debug("data size: %d",len(p.Data()))

		}
		gmf.Release(srcFrame)
	}

}
// close encoder
func (e *FFmpegEncoder) Close() {
	e.wg.Done()
}
