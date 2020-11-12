package sqhttp

import (
	"io"
	"net/http"
)

type (
	flusherPusherCloseNotifierHijackerReaderFromStringWriter struct {
		http.ResponseWriter
		FlusherPusherCloseNotifierHijackerReaderFromStringWriter
	}

	FlusherPusherCloseNotifierHijackerReaderFromStringWriter interface {
		http.Flusher
		http.Pusher
		http.CloseNotifier
		http.Hijacker
		io.ReaderFrom
		io.StringWriter
	}

	pusherCloseNotifierHijackerReaderFromStringWriter struct {
		http.ResponseWriter
		PusherCloseNotifierHijackerReaderFromStringWriter
	}

	PusherCloseNotifierHijackerReaderFromStringWriter interface {
		http.Pusher
		http.CloseNotifier
		http.Hijacker
		io.ReaderFrom
		io.StringWriter
	}

	flusherCloseNotifierHijackerReaderFromStringWriter struct {
		http.ResponseWriter
		FlusherCloseNotifierHijackerReaderFromStringWriter
	}

	FlusherCloseNotifierHijackerReaderFromStringWriter interface {
		http.Flusher
		http.CloseNotifier
		http.Hijacker
		io.ReaderFrom
		io.StringWriter
	}

	flusherPusherCloseNotifierHijackerStringWriter struct {
		http.ResponseWriter
		FlusherPusherCloseNotifierHijackerStringWriter
	}

	FlusherPusherCloseNotifierHijackerStringWriter interface {
		http.Flusher
		http.Pusher
		http.CloseNotifier
		http.Hijacker
		io.StringWriter
	}

	flusherPusherHijackerReaderFromStringWriter struct {
		http.ResponseWriter
		FlusherPusherHijackerReaderFromStringWriter
	}

	FlusherPusherHijackerReaderFromStringWriter interface {
		http.Flusher
		http.Pusher
		http.Hijacker
		io.ReaderFrom
		io.StringWriter
	}

	flusherPusherCloseNotifierReaderFromStringWriter struct {
		http.ResponseWriter
		FlusherPusherCloseNotifierReaderFromStringWriter
	}

	FlusherPusherCloseNotifierReaderFromStringWriter interface {
		http.Flusher
		http.Pusher
		http.CloseNotifier
		io.ReaderFrom
		io.StringWriter
	}

	flusherPusherCloseNotifierHijackerReaderFrom struct {
		http.ResponseWriter
		FlusherPusherCloseNotifierHijackerReaderFrom
	}

	FlusherPusherCloseNotifierHijackerReaderFrom interface {
		http.Flusher
		http.Pusher
		http.CloseNotifier
		http.Hijacker
		io.ReaderFrom
	}

	flusherPusherCloseNotifierReaderFrom struct {
		http.ResponseWriter
		FlusherPusherCloseNotifierReaderFrom
	}

	FlusherPusherCloseNotifierReaderFrom interface {
		http.Flusher
		http.Pusher
		http.CloseNotifier
		io.ReaderFrom
	}

	flusherHijackerReaderFromStringWriter struct {
		http.ResponseWriter
		FlusherHijackerReaderFromStringWriter
	}

	FlusherHijackerReaderFromStringWriter interface {
		http.Flusher
		http.Hijacker
		io.ReaderFrom
		io.StringWriter
	}

	flusherPusherCloseNotifierHijacker struct {
		http.ResponseWriter
		FlusherPusherCloseNotifierHijacker
	}

	FlusherPusherCloseNotifierHijacker interface {
		http.Flusher
		http.Pusher
		http.CloseNotifier
		http.Hijacker
	}

	flusherPusherHijackerStringWriter struct {
		http.ResponseWriter
		FlusherPusherHijackerStringWriter
	}

	FlusherPusherHijackerStringWriter interface {
		http.Flusher
		http.Pusher
		http.Hijacker
		io.StringWriter
	}

	pusherHijackerReaderFromStringWriter struct {
		http.ResponseWriter
		PusherHijackerReaderFromStringWriter
	}

	PusherHijackerReaderFromStringWriter interface {
		http.Pusher
		http.Hijacker
		io.ReaderFrom
		io.StringWriter
	}

	pusherCloseNotifierHijackerStringWriter struct {
		http.ResponseWriter
		PusherCloseNotifierHijackerStringWriter
	}

	PusherCloseNotifierHijackerStringWriter interface {
		http.Pusher
		http.CloseNotifier
		http.Hijacker
		io.StringWriter
	}

	closeNotifierHijackerReaderFromStringWriter struct {
		http.ResponseWriter
		CloseNotifierHijackerReaderFromStringWriter
	}

	CloseNotifierHijackerReaderFromStringWriter interface {
		http.CloseNotifier
		http.Hijacker
		io.ReaderFrom
		io.StringWriter
	}

	pusherCloseNotifierReaderFromStringWriter struct {
		http.ResponseWriter
		PusherCloseNotifierReaderFromStringWriter
	}

	PusherCloseNotifierReaderFromStringWriter interface {
		http.Pusher
		http.CloseNotifier
		io.ReaderFrom
		io.StringWriter
	}

	flusherCloseNotifierReaderFromStringWriter struct {
		http.ResponseWriter
		FlusherCloseNotifierReaderFromStringWriter
	}

	FlusherCloseNotifierReaderFromStringWriter interface {
		http.Flusher
		http.CloseNotifier
		io.ReaderFrom
		io.StringWriter
	}

	pusherCloseNotifierHijackerReaderFrom struct {
		http.ResponseWriter
		PusherCloseNotifierHijackerReaderFrom
	}

	PusherCloseNotifierHijackerReaderFrom interface {
		http.Pusher
		http.CloseNotifier
		http.Hijacker
		io.ReaderFrom
	}

	flusherPusherReaderFromStringWriter struct {
		http.ResponseWriter
		FlusherPusherReaderFromStringWriter
	}

	FlusherPusherReaderFromStringWriter interface {
		http.Flusher
		http.Pusher
		io.ReaderFrom
		io.StringWriter
	}

	flusherCloseNotifierHijackerReaderFrom struct {
		http.ResponseWriter
		FlusherCloseNotifierHijackerReaderFrom
	}

	FlusherCloseNotifierHijackerReaderFrom interface {
		http.Flusher
		http.CloseNotifier
		http.Hijacker
		io.ReaderFrom
	}

	flusherPusherHijackerReaderFrom struct {
		http.ResponseWriter
		FlusherPusherHijackerReaderFrom
	}

	FlusherPusherHijackerReaderFrom interface {
		http.Flusher
		http.Pusher
		http.Hijacker
		io.ReaderFrom
	}

	flusherCloseNotifierHijackerStringWriter struct {
		http.ResponseWriter
		FlusherCloseNotifierHijackerStringWriter
	}

	FlusherCloseNotifierHijackerStringWriter interface {
		http.Flusher
		http.CloseNotifier
		http.Hijacker
		io.StringWriter
	}

	flusherPusherCloseNotifierStringWriter struct {
		http.ResponseWriter
		FlusherPusherCloseNotifierStringWriter
	}

	FlusherPusherCloseNotifierStringWriter interface {
		http.Flusher
		http.Pusher
		http.CloseNotifier
		io.StringWriter
	}

	flusherCloseNotifierReaderFrom struct {
		http.ResponseWriter
		FlusherCloseNotifierReaderFrom
	}

	FlusherCloseNotifierReaderFrom interface {
		http.Flusher
		http.CloseNotifier
		io.ReaderFrom
	}

	flusherReaderFromStringWriter struct {
		http.ResponseWriter
		FlusherReaderFromStringWriter
	}

	FlusherReaderFromStringWriter interface {
		http.Flusher
		io.ReaderFrom
		io.StringWriter
	}

	pusherCloseNotifierReaderFrom struct {
		http.ResponseWriter
		PusherCloseNotifierReaderFrom
	}

	PusherCloseNotifierReaderFrom interface {
		http.Pusher
		http.CloseNotifier
		io.ReaderFrom
	}

	pusherHijackerReaderFrom struct {
		http.ResponseWriter
		PusherHijackerReaderFrom
	}

	PusherHijackerReaderFrom interface {
		http.Pusher
		http.Hijacker
		io.ReaderFrom
	}

	pusherReaderFromStringWriter struct {
		http.ResponseWriter
		PusherReaderFromStringWriter
	}

	PusherReaderFromStringWriter interface {
		http.Pusher
		io.ReaderFrom
		io.StringWriter
	}

	closeNotifierHijackerReaderFrom struct {
		http.ResponseWriter
		CloseNotifierHijackerReaderFrom
	}

	CloseNotifierHijackerReaderFrom interface {
		http.CloseNotifier
		http.Hijacker
		io.ReaderFrom
	}

	flusherPusherReaderFrom struct {
		http.ResponseWriter
		FlusherPusherReaderFrom
	}

	FlusherPusherReaderFrom interface {
		http.Flusher
		http.Pusher
		io.ReaderFrom
	}

	closeNotifierReaderFromStringWriter struct {
		http.ResponseWriter
		CloseNotifierReaderFromStringWriter
	}

	CloseNotifierReaderFromStringWriter interface {
		http.CloseNotifier
		io.ReaderFrom
		io.StringWriter
	}

	flusherHijackerStringWriter struct {
		http.ResponseWriter
		FlusherHijackerStringWriter
	}

	FlusherHijackerStringWriter interface {
		http.Flusher
		http.Hijacker
		io.StringWriter
	}

	flusherHijackerReaderFrom struct {
		http.ResponseWriter
		FlusherHijackerReaderFrom
	}

	FlusherHijackerReaderFrom interface {
		http.Flusher
		http.Hijacker
		io.ReaderFrom
	}

	pusherCloseNotifierHijacker struct {
		http.ResponseWriter
		PusherCloseNotifierHijacker
	}

	PusherCloseNotifierHijacker interface {
		http.Pusher
		http.CloseNotifier
		http.Hijacker
	}

	flusherCloseNotifierHijacker struct {
		http.ResponseWriter
		FlusherCloseNotifierHijacker
	}

	FlusherCloseNotifierHijacker interface {
		http.Flusher
		http.CloseNotifier
		http.Hijacker
	}

	flusherPusherStringWriter struct {
		http.ResponseWriter
		FlusherPusherStringWriter
	}

	FlusherPusherStringWriter interface {
		http.Flusher
		http.Pusher
		io.StringWriter
	}

	flusherPusherHijacker struct {
		http.ResponseWriter
		FlusherPusherHijacker
	}

	FlusherPusherHijacker interface {
		http.Flusher
		http.Pusher
		http.Hijacker
	}

	flusherCloseNotifierStringWriter struct {
		http.ResponseWriter
		FlusherCloseNotifierStringWriter
	}

	FlusherCloseNotifierStringWriter interface {
		http.Flusher
		http.CloseNotifier
		io.StringWriter
	}

	pusherCloseNotifierStringWriter struct {
		http.ResponseWriter
		PusherCloseNotifierStringWriter
	}

	PusherCloseNotifierStringWriter interface {
		http.Pusher
		http.CloseNotifier
		io.StringWriter
	}

	closeNotifierHijackerStringWriter struct {
		http.ResponseWriter
		CloseNotifierHijackerStringWriter
	}

	CloseNotifierHijackerStringWriter interface {
		http.CloseNotifier
		http.Hijacker
		io.StringWriter
	}

	flusherPusherCloseNotifier struct {
		http.ResponseWriter
		FlusherPusherCloseNotifier
	}

	FlusherPusherCloseNotifier interface {
		http.Flusher
		http.Pusher
		http.CloseNotifier
	}

	pusherHijackerStringWriter struct {
		http.ResponseWriter
		PusherHijackerStringWriter
	}

	PusherHijackerStringWriter interface {
		http.Pusher
		http.Hijacker
		io.StringWriter
	}

	hijackerReaderFromStringWriter struct {
		http.ResponseWriter
		HijackerReaderFromStringWriter
	}

	HijackerReaderFromStringWriter interface {
		http.Hijacker
		io.ReaderFrom
		io.StringWriter
	}

	pusherCloseNotifier struct {
		http.ResponseWriter
		PusherCloseNotifier
	}

	PusherCloseNotifier interface {
		http.Pusher
		http.CloseNotifier
	}

	flusherPusher struct {
		http.ResponseWriter
		FlusherPusher
	}

	FlusherPusher interface {
		http.Flusher
		http.Pusher
	}

	closeNotifierStringWriter struct {
		http.ResponseWriter
		CloseNotifierStringWriter
	}

	CloseNotifierStringWriter interface {
		http.CloseNotifier
		io.StringWriter
	}

	pusherStringWriter struct {
		http.ResponseWriter
		PusherStringWriter
	}

	PusherStringWriter interface {
		http.Pusher
		io.StringWriter
	}

	flusherStringWriter struct {
		http.ResponseWriter
		FlusherStringWriter
	}

	FlusherStringWriter interface {
		http.Flusher
		io.StringWriter
	}

	readerFromStringWriter struct {
		http.ResponseWriter
		ReaderFromStringWriter
	}

	ReaderFromStringWriter interface {
		io.ReaderFrom
		io.StringWriter
	}

	hijackerReaderFrom struct {
		http.ResponseWriter
		HijackerReaderFrom
	}

	HijackerReaderFrom interface {
		http.Hijacker
		io.ReaderFrom
	}

	closeNotifierReaderFrom struct {
		http.ResponseWriter
		CloseNotifierReaderFrom
	}

	CloseNotifierReaderFrom interface {
		http.CloseNotifier
		io.ReaderFrom
	}

	pusherReaderFrom struct {
		http.ResponseWriter
		PusherReaderFrom
	}

	PusherReaderFrom interface {
		http.Pusher
		io.ReaderFrom
	}

	flusherReaderFrom struct {
		http.ResponseWriter
		FlusherReaderFrom
	}

	FlusherReaderFrom interface {
		http.Flusher
		io.ReaderFrom
	}

	hijackerStringWriter struct {
		http.ResponseWriter
		HijackerStringWriter
	}

	HijackerStringWriter interface {
		http.Hijacker
		io.StringWriter
	}

	flusherCloseNotifier struct {
		http.ResponseWriter
		FlusherCloseNotifier
	}

	FlusherCloseNotifier interface {
		http.Flusher
		http.CloseNotifier
	}

	closeNotifierHijacker struct {
		http.ResponseWriter
		CloseNotifierHijacker
	}

	CloseNotifierHijacker interface {
		http.CloseNotifier
		http.Hijacker
	}

	pusherHijacker struct {
		http.ResponseWriter
		PusherHijacker
	}

	PusherHijacker interface {
		http.Pusher
		http.Hijacker
	}

	flusherHijacker struct {
		http.ResponseWriter
		FlusherHijacker
	}

	FlusherHijacker interface {
		http.Flusher
		http.Hijacker
	}

	readerFrom struct {
		http.ResponseWriter
		ReaderFrom
	}

	ReaderFrom interface {
		io.ReaderFrom
	}

	flusher struct {
		http.ResponseWriter
		Flusher
	}

	Flusher interface {
		http.Flusher
	}

	closeNotifier struct {
		http.ResponseWriter
		CloseNotifier
	}

	CloseNotifier interface {
		http.CloseNotifier
	}

	stringWriter struct {
		http.ResponseWriter
		StringWriter
	}

	StringWriter interface {
		io.StringWriter
	}

	pusher struct {
		http.ResponseWriter
		Pusher
	}

	Pusher interface {
		http.Pusher
	}

	hijacker struct {
		http.ResponseWriter
		Hijacker
	}

	Hijacker interface {
		http.Hijacker
	}
)

func adaptResponseWriter(wrapper, wrapped http.ResponseWriter) http.ResponseWriter {
	switch actual := wrapped.(type) {

	case FlusherPusherCloseNotifierHijackerReaderFromStringWriter:
		return flusherPusherCloseNotifierHijackerReaderFromStringWriter{
			ResponseWriter: wrapper,
			FlusherPusherCloseNotifierHijackerReaderFromStringWriter: actual,
		}

	case PusherCloseNotifierHijackerReaderFromStringWriter:
		return pusherCloseNotifierHijackerReaderFromStringWriter{
			ResponseWriter: wrapper,
			PusherCloseNotifierHijackerReaderFromStringWriter: actual,
		}

	case FlusherCloseNotifierHijackerReaderFromStringWriter:
		return flusherCloseNotifierHijackerReaderFromStringWriter{
			ResponseWriter: wrapper,
			FlusherCloseNotifierHijackerReaderFromStringWriter: actual,
		}

	case FlusherPusherCloseNotifierHijackerStringWriter:
		return flusherPusherCloseNotifierHijackerStringWriter{
			ResponseWriter: wrapper,
			FlusherPusherCloseNotifierHijackerStringWriter: actual,
		}

	case FlusherPusherHijackerReaderFromStringWriter:
		return flusherPusherHijackerReaderFromStringWriter{
			ResponseWriter: wrapper,
			FlusherPusherHijackerReaderFromStringWriter: actual,
		}

	case FlusherPusherCloseNotifierReaderFromStringWriter:
		return flusherPusherCloseNotifierReaderFromStringWriter{
			ResponseWriter: wrapper,
			FlusherPusherCloseNotifierReaderFromStringWriter: actual,
		}

	case FlusherPusherCloseNotifierHijackerReaderFrom:
		return flusherPusherCloseNotifierHijackerReaderFrom{
			ResponseWriter: wrapper,
			FlusherPusherCloseNotifierHijackerReaderFrom: actual,
		}

	case FlusherPusherCloseNotifierReaderFrom:
		return flusherPusherCloseNotifierReaderFrom{
			ResponseWriter:                       wrapper,
			FlusherPusherCloseNotifierReaderFrom: actual,
		}

	case FlusherHijackerReaderFromStringWriter:
		return flusherHijackerReaderFromStringWriter{
			ResponseWriter:                        wrapper,
			FlusherHijackerReaderFromStringWriter: actual,
		}

	case FlusherPusherCloseNotifierHijacker:
		return flusherPusherCloseNotifierHijacker{
			ResponseWriter:                     wrapper,
			FlusherPusherCloseNotifierHijacker: actual,
		}

	case FlusherPusherHijackerStringWriter:
		return flusherPusherHijackerStringWriter{
			ResponseWriter:                    wrapper,
			FlusherPusherHijackerStringWriter: actual,
		}

	case PusherHijackerReaderFromStringWriter:
		return pusherHijackerReaderFromStringWriter{
			ResponseWriter:                       wrapper,
			PusherHijackerReaderFromStringWriter: actual,
		}

	case PusherCloseNotifierHijackerStringWriter:
		return pusherCloseNotifierHijackerStringWriter{
			ResponseWriter:                          wrapper,
			PusherCloseNotifierHijackerStringWriter: actual,
		}

	case CloseNotifierHijackerReaderFromStringWriter:
		return closeNotifierHijackerReaderFromStringWriter{
			ResponseWriter: wrapper,
			CloseNotifierHijackerReaderFromStringWriter: actual,
		}

	case PusherCloseNotifierReaderFromStringWriter:
		return pusherCloseNotifierReaderFromStringWriter{
			ResponseWriter: wrapper,
			PusherCloseNotifierReaderFromStringWriter: actual,
		}

	case FlusherCloseNotifierReaderFromStringWriter:
		return flusherCloseNotifierReaderFromStringWriter{
			ResponseWriter: wrapper,
			FlusherCloseNotifierReaderFromStringWriter: actual,
		}

	case PusherCloseNotifierHijackerReaderFrom:
		return pusherCloseNotifierHijackerReaderFrom{
			ResponseWriter:                        wrapper,
			PusherCloseNotifierHijackerReaderFrom: actual,
		}

	case FlusherPusherReaderFromStringWriter:
		return flusherPusherReaderFromStringWriter{
			ResponseWriter:                      wrapper,
			FlusherPusherReaderFromStringWriter: actual,
		}

	case FlusherCloseNotifierHijackerReaderFrom:
		return flusherCloseNotifierHijackerReaderFrom{
			ResponseWriter:                         wrapper,
			FlusherCloseNotifierHijackerReaderFrom: actual,
		}

	case FlusherPusherHijackerReaderFrom:
		return flusherPusherHijackerReaderFrom{
			ResponseWriter:                  wrapper,
			FlusherPusherHijackerReaderFrom: actual,
		}

	case FlusherCloseNotifierHijackerStringWriter:
		return flusherCloseNotifierHijackerStringWriter{
			ResponseWriter:                           wrapper,
			FlusherCloseNotifierHijackerStringWriter: actual,
		}

	case FlusherPusherCloseNotifierStringWriter:
		return flusherPusherCloseNotifierStringWriter{
			ResponseWriter:                         wrapper,
			FlusherPusherCloseNotifierStringWriter: actual,
		}

	case FlusherCloseNotifierReaderFrom:
		return flusherCloseNotifierReaderFrom{
			ResponseWriter:                 wrapper,
			FlusherCloseNotifierReaderFrom: actual,
		}

	case FlusherReaderFromStringWriter:
		return flusherReaderFromStringWriter{
			ResponseWriter:                wrapper,
			FlusherReaderFromStringWriter: actual,
		}

	case PusherCloseNotifierReaderFrom:
		return pusherCloseNotifierReaderFrom{
			ResponseWriter:                wrapper,
			PusherCloseNotifierReaderFrom: actual,
		}

	case PusherHijackerReaderFrom:
		return pusherHijackerReaderFrom{
			ResponseWriter:           wrapper,
			PusherHijackerReaderFrom: actual,
		}

	case PusherReaderFromStringWriter:
		return pusherReaderFromStringWriter{
			ResponseWriter:               wrapper,
			PusherReaderFromStringWriter: actual,
		}

	case CloseNotifierHijackerReaderFrom:
		return closeNotifierHijackerReaderFrom{
			ResponseWriter:                  wrapper,
			CloseNotifierHijackerReaderFrom: actual,
		}

	case FlusherPusherReaderFrom:
		return flusherPusherReaderFrom{
			ResponseWriter:          wrapper,
			FlusherPusherReaderFrom: actual,
		}

	case CloseNotifierReaderFromStringWriter:
		return closeNotifierReaderFromStringWriter{
			ResponseWriter:                      wrapper,
			CloseNotifierReaderFromStringWriter: actual,
		}

	case FlusherHijackerStringWriter:
		return flusherHijackerStringWriter{
			ResponseWriter:              wrapper,
			FlusherHijackerStringWriter: actual,
		}

	case FlusherHijackerReaderFrom:
		return flusherHijackerReaderFrom{
			ResponseWriter:            wrapper,
			FlusherHijackerReaderFrom: actual,
		}

	case PusherCloseNotifierHijacker:
		return pusherCloseNotifierHijacker{
			ResponseWriter:              wrapper,
			PusherCloseNotifierHijacker: actual,
		}

	case FlusherCloseNotifierHijacker:
		return flusherCloseNotifierHijacker{
			ResponseWriter:               wrapper,
			FlusherCloseNotifierHijacker: actual,
		}

	case FlusherPusherStringWriter:
		return flusherPusherStringWriter{
			ResponseWriter:            wrapper,
			FlusherPusherStringWriter: actual,
		}

	case FlusherPusherHijacker:
		return flusherPusherHijacker{
			ResponseWriter:        wrapper,
			FlusherPusherHijacker: actual,
		}

	case FlusherCloseNotifierStringWriter:
		return flusherCloseNotifierStringWriter{
			ResponseWriter:                   wrapper,
			FlusherCloseNotifierStringWriter: actual,
		}

	case PusherCloseNotifierStringWriter:
		return pusherCloseNotifierStringWriter{
			ResponseWriter:                  wrapper,
			PusherCloseNotifierStringWriter: actual,
		}

	case CloseNotifierHijackerStringWriter:
		return closeNotifierHijackerStringWriter{
			ResponseWriter:                    wrapper,
			CloseNotifierHijackerStringWriter: actual,
		}

	case FlusherPusherCloseNotifier:
		return flusherPusherCloseNotifier{
			ResponseWriter:             wrapper,
			FlusherPusherCloseNotifier: actual,
		}

	case PusherHijackerStringWriter:
		return pusherHijackerStringWriter{
			ResponseWriter:             wrapper,
			PusherHijackerStringWriter: actual,
		}

	case HijackerReaderFromStringWriter:
		return hijackerReaderFromStringWriter{
			ResponseWriter:                 wrapper,
			HijackerReaderFromStringWriter: actual,
		}

	case PusherCloseNotifier:
		return pusherCloseNotifier{
			ResponseWriter:      wrapper,
			PusherCloseNotifier: actual,
		}

	case FlusherPusher:
		return flusherPusher{
			ResponseWriter: wrapper,
			FlusherPusher:  actual,
		}

	case CloseNotifierStringWriter:
		return closeNotifierStringWriter{
			ResponseWriter:            wrapper,
			CloseNotifierStringWriter: actual,
		}

	case PusherStringWriter:
		return pusherStringWriter{
			ResponseWriter:     wrapper,
			PusherStringWriter: actual,
		}

	case FlusherStringWriter:
		return flusherStringWriter{
			ResponseWriter:      wrapper,
			FlusherStringWriter: actual,
		}

	case ReaderFromStringWriter:
		return readerFromStringWriter{
			ResponseWriter:         wrapper,
			ReaderFromStringWriter: actual,
		}

	case HijackerReaderFrom:
		return hijackerReaderFrom{
			ResponseWriter:     wrapper,
			HijackerReaderFrom: actual,
		}

	case CloseNotifierReaderFrom:
		return closeNotifierReaderFrom{
			ResponseWriter:          wrapper,
			CloseNotifierReaderFrom: actual,
		}

	case PusherReaderFrom:
		return pusherReaderFrom{
			ResponseWriter:   wrapper,
			PusherReaderFrom: actual,
		}

	case FlusherReaderFrom:
		return flusherReaderFrom{
			ResponseWriter:    wrapper,
			FlusherReaderFrom: actual,
		}

	case HijackerStringWriter:
		return hijackerStringWriter{
			ResponseWriter:       wrapper,
			HijackerStringWriter: actual,
		}

	case FlusherCloseNotifier:
		return flusherCloseNotifier{
			ResponseWriter:       wrapper,
			FlusherCloseNotifier: actual,
		}

	case CloseNotifierHijacker:
		return closeNotifierHijacker{
			ResponseWriter:        wrapper,
			CloseNotifierHijacker: actual,
		}

	case PusherHijacker:
		return pusherHijacker{
			ResponseWriter: wrapper,
			PusherHijacker: actual,
		}

	case FlusherHijacker:
		return flusherHijacker{
			ResponseWriter:  wrapper,
			FlusherHijacker: actual,
		}

	case ReaderFrom:
		return readerFrom{
			ResponseWriter: wrapper,
			ReaderFrom:     actual,
		}

	case Flusher:
		return flusher{
			ResponseWriter: wrapper,
			Flusher:        actual,
		}

	case CloseNotifier:
		return closeNotifier{
			ResponseWriter: wrapper,
			CloseNotifier:  actual,
		}

	case StringWriter:
		return stringWriter{
			ResponseWriter: wrapper,
			StringWriter:   actual,
		}

	case Pusher:
		return pusher{
			ResponseWriter: wrapper,
			Pusher:         actual,
		}

	case Hijacker:
		return hijacker{
			ResponseWriter: wrapper,
			Hijacker:       actual,
		}

	default:
		return wrapper
	}
}
