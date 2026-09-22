package runner

import (
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogStream_StepEndFlushesLabelAndOutputAsOneBatch(t *testing.T) {
	rec := &frameRecorder{}
	l := newLogStream("d1", rec.send)
	logf := l.step(LogPhaseCheckout, "clone org/app@main")
	logf("Cloning into '/data/repo'...\n")
	logf("done\n")
	assert.Empty(t, rec.all(), "short output waits for the step to end")

	l.flush()
	frames := rec.all()
	require.Len(t, frames, 1)
	assert.Equal(t, Frame{Type: FrameDeployLog, ID: "d1", Phase: "checkout", Log: "clone org/app@main\nCloning into '/data/repo'...\ndone\n", TS: frames[0].TS}, frames[0])
	assert.Positive(t, frames[0].TS)
	assert.NoError(t, frames[0].Validate())

	l.flush()
	assert.Len(t, rec.all(), 1, "an empty buffer sends nothing")
}

func TestLogStream_QuietCommandFlushesAfterDelay(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		rec := &frameRecorder{}
		l := newLogStream("d1", rec.send)
		logf := l.step(LogPhaseBuild, "docker build api:b1")
		logf("Step 1/4 : FROM golang\n")

		time.Sleep(logFlushAfter - time.Millisecond)
		synctest.Wait()
		assert.Empty(t, rec.all())

		time.Sleep(time.Millisecond)
		synctest.Wait()
		require.Len(t, rec.all(), 1)
		assert.Equal(t, "docker build api:b1\nStep 1/4 : FROM golang\n", rec.all()[0].Log)

		logf("Step 2/4 : COPY . .\n")
		time.Sleep(logFlushAfter)
		synctest.Wait()
		require.Len(t, rec.all(), 2, "the timer re-arms for the next batch")
		assert.Equal(t, "Step 2/4 : COPY . .\n", rec.all()[1].Log)
	})
}

func TestLogStream_LargeOutputFlushesAtByteThreshold(t *testing.T) {
	rec := &frameRecorder{}
	l := newLogStream("d1", rec.send)
	logf := l.step(LogPhaseBuild, "docker build")
	line := strings.Repeat("x", 1023) + "\n"
	for range logFlushBytes / len(line) {
		logf(line)
	}
	require.Len(t, rec.all(), 1)
	assert.GreaterOrEqual(t, len(rec.all()[0].Log), logFlushBytes)
}

func TestLogStream_PhaseChangeSplitsBatches(t *testing.T) {
	rec := &frameRecorder{}
	l := newLogStream("d1", rec.send)
	l.step(LogPhaseBuild, "docker build")("ok\n")
	l.step(LogPhaseDeploy, "docker run")
	l.close()

	frames := rec.all()
	require.Len(t, frames, 2)
	assert.Equal(t, "build", frames[0].Phase)
	assert.Equal(t, "docker build\nok\n", frames[0].Log)
	assert.Equal(t, "deploy", frames[1].Phase)
	assert.Equal(t, "docker run\n", frames[1].Log)
}

func TestLogStream_CapsTotalBytesWithTruncationLine(t *testing.T) {
	rec := &frameRecorder{}
	l := newLogStream("d1", rec.send)
	logf := l.step(LogPhaseBuild, "docker build")
	chunk := strings.Repeat("y", 4095) + "\n"
	for range logMaxBytes/len(chunk) + 2 {
		logf(chunk)
	}
	logf("after the cap\n")
	l.close()

	var total int
	for _, f := range rec.all() {
		total += len(f.Log)
		assert.NotContains(t, f.Log, "after the cap")
	}
	assert.LessOrEqual(t, total, logMaxBytes+len(logTruncatedLine))
	assert.True(t, strings.HasSuffix(rec.last().Log, logTruncatedLine), "the last batch ends with the truncation marker")
	assert.Equal(t, 1, strings.Count(rec.logText(LogPhaseBuild), logTruncatedLine))
}

func TestLogStream_CloseFlushesAndStopsTheTimer(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		rec := &frameRecorder{}
		l := newLogStream("d1", rec.send)
		l.step(LogPhaseDeploy, "docker pull img")("pulled\n")
		l.close()
		require.Len(t, rec.all(), 1)

		l.step(LogPhaseDeploy, "late")("dropped\n")
		time.Sleep(2 * logFlushAfter)
		synctest.Wait()
		assert.Len(t, rec.all(), 1, "a write after close never sends")
	})
}

func TestLogStream_EmptyWriteIsIgnored(t *testing.T) {
	rec := &frameRecorder{}
	l := newLogStream("d1", rec.send)
	l.write(LogPhaseDeploy, "")
	l.close()
	assert.Empty(t, rec.all())
}
