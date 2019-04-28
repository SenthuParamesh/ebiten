// Copyright 2018 The Ebiten Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package audio_test

import (
	"runtime"
	"testing"
	"time"

	. "github.com/hajimehoshi/ebiten/audio"
)

var context *Context

func init() {
	var err error
	context, err = NewContext(44100)
	if err != nil {
		panic(err)
	}
}

// Issue #746
func TestGC(t *testing.T) {
	p, _ := NewPlayer(context, BytesReadSeekCloser(make([]byte, 4)))
	got := PlayersNumForTesting()
	if want := 0; got != want {
		t.Errorf("PlayersNum(): got: %d, want: %d", got, want)
	}

	p.Play()
	got = PlayersNumForTesting()
	if want := 1; got != want {
		t.Errorf("PlayersNum() after Play: got: %d, want: %d", got, want)
	}

	runtime.KeepAlive(p)
	p = nil
	runtime.GC()

	for i := 0; i < 10; i++ {
		got = PlayersNumForTesting()
		if want := 0; got == want {
			return
		}
		if err := UpdateForTesting(); err != nil {
			t.Error(err)
		}
		// 200[ms] should be enough all the bytes are consumed.
		// TODO: This is a darty hack. Would it be possible to use virtual time?
		time.Sleep(200 * time.Millisecond)
	}
	t.Errorf("time out")
}
