package main

import (
	"os"
	"time"

	"github.com/warpfork/go-ansicraft"
)

func main() {
	os.Stdout.Write([]byte("output before controller\n"))

	term := ansicraft.NewController(os.Stdout)
	term.Write([]byte("controlled output begins\n"))
	pause()

	term.SetTrailer([][]byte{
		[]byte("^^^^^^^^"),
		[]byte(" -> trailer line 2"),
	})
	pause()

	term.Write([]byte("this is a whole line\n"))
	pause()
	term.Write([]byte("all at once\n"))
	pause()

	term.Write([]byte("this line takes"))
	pause()
	term.Write([]byte("... \x1B[32msome time"))
	pause()
	term.Write([]byte("."))
	pause()
	term.Write([]byte("."))
	pause()
	term.Write([]byte(".\n"))
	pause()

	term.Write([]byte("plain scrollback\n"))
	pause()

	term.SetTrailer([][]byte{
		[]byte("^^^^^^^^"),
		[]byte(" -> trailer line 2"),
		[]byte(" -> trailer line longer"),
		[]byte(" -> and line 3"),
	})
	term.Write([]byte("more plain scrollback\n"))
	pause()

	term.Write([]byte("even more plain scrollback\n"))
	pause()

	term.SetTrailer([][]byte{
		[]byte("^^^^^^^^"),
		[]byte(" -> trailer shorter now"),
	})
	term.Write([]byte("yet more plain scrollback\n"))
	pause()

	term.Write([]byte("here's two\n lines in one write\n"))
	pause()

	term.Write([]byte("here's one line and\n a partial... "))
	pause()

	term.Write([]byte("... done\nwith another full, too.\n"))
	pause()

	term.Write([]byte("edge case test: several breaks in a row\n\nsurvived?"))
	pause()

	term.Write([]byte("...hope so.\n\nshould work regardless of if the last line was partial, too.\n"))
	pause()

	term.Write([]byte("next i'm gonna nil the trailer entirely\n"))
	pause()

	term.SetTrailer(nil)
	pause()

	term.Write([]byte("signing off\n"))
	pause()
}

func pause() {
	time.Sleep(1000 * time.Millisecond)
}
