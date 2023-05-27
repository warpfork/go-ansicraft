package ansicraft

import (
	"bytes"
	"io"
	"strconv"
	"sync"
)

/*

IMPORTANT ANSI TERMINAL CODES

Getting clear space:

- `CSI 0 J` -- clear from cursor to end of screen.

Relative motion of the cursor:

- `\r` -- move cursor to start of current line.
- `CSI n A` -- move the cursor up `n` lines.
- `CSI n F` -- move the cursor up `n` lines and to the start of the line -- but it's probably more widely supported to compose 'A' and '\r'.

There are also mechanisms for directly asking the terminal to
store and restore cursor positions:

- `CSI s` -- Save.
- `CSI u` -- Restore.
- `CSI 7` -- also Save, but DEC style.
- `CSI 8` -- also Restore, but DEC style.

Unfortunately, in terminals I've tested, the save/restore codes
don't result in reliable operation in the presence of unexpected linebreaks from wrapping.
In those situations, usually, the restore command gets you to the correct column,
but appears to do nothing at all to the row.
(This challenge isn't entirely unique to the save/restore feature -- really, any approaches
to managing the position of the terminal's cursor will become hard to control in the case of
any cursor shifts outside of our controlling.
There's no possible approach to managing this other than "make sure your content doesn't do that" --
either by having entirely known content, or making sure your content is stripped of the relevant codes.
However, I find the predictability of the cursor restore feature to be actually somewhat worse
than the predictability of relative move instructions, which still at least reliably do what they say
even if the cursor has been moved by external forces.)

The save/restore features are also documented as extensions,
and *may* be less widely supported than other approaches like the "A" code.
In practice, I've found them to generally exist in terminals I've tested...
but also to have a greater likelihood of edge cases, some of which I don't understand at all.

You can test some of this in bash with commands like this:

	echo -en "heyo\033[smorewords\033[uwow\n"
	echo -en "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA\nBBBBBBBB\033[sBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB\nCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCC\n\033[uxxxxxxxxxxxxxxxxx\n"
	echo -en "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA\nBBBBBBBB\033[sBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB\nCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCC\n\033[u\033[0Jxxxxxxxxxxxxxxxxx\n"
	echo -en "AAAAAAAAAAA\r\033[0Jxxx\n"

Mind what happens if you make the terminal window narrower than the length of the lines; therein, there be dragons.

...

What did we actually do in the code below?

The r-A-J approach.

I tried the save/restore cursor approach first.  But I find that it just isn't working in practice.
I'm unclear on why.  In a terminals I test with the above bash, it can work,
and yet when I attempt to use the go code below (which is now commented out),
it's not operating correctly at all.
(I've left the functions for save-restore where I think they belong, but commented
out the actions itself, in case anyone wants to give a crack at this in the future;
it's very similar, but not quite exactly the same, as where the cursorMoveUp calls are.)

The code below now uses the r-A-J sequence.
This requires keeping track of its expected jump sizes in order to do so;
however, this is a pretty mild effort.  (Considering that we are already
necessarily keeping buffers for repaining the trailer content after new scrollback,
asking the size of it is negligible additional work.)
This seems to work quite solidly in practice.

*/

// Controller wraps an `io.Writer` and treats it like a terminal, and allows "trailer" content to be attached.
// The trailer content is always rendered at the bttom of the output, and other output written to this Controller
// is rendered as scrollback above the trailer.
//
// To update the trailer, call `SetTrailer`.
// To write content to the scrollback, just treat the Controller as a regular `io.Writer`.
type Controller struct {
	wr io.Writer // Should be a terminal device.  But largely, we don't actually care much -- we'll emit ANSI regardless.

	mu sync.Mutex // Writes to scrollback and changes to the trailer must not be emitted concurrently.

	partial bytes.Buffer // A line of regular output that hasn't received a trailing linebreak yet.  We print it (and reprint over it) almost as if it's part of the trailer until it gets a real linebreak.

	trailer [][]byte // Broken by lines.  Reprint this every time the scrollback advances.  The length of this is also how much we have to back up and clear whenever repainting or when writing a regular scrollback line.
}

func NewController(tty io.Writer) *Controller {
	tc := &Controller{wr: tty}
	tc.cursorPositionSave()
	return tc
}

func (tc *Controller) SetTrailer(lines [][]byte) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	tc.trailer = lines
	tc.clearToEnd()
	tc.printTrailer()
}

func (tc *Controller) Write(msg []byte) (int, error) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	// The cursor should already be at the start of the previous trailer.
	// (We parked it here already, for anti-fragility purposes, in case any other uncontrolled writes reached the terminal without our interception.)
	// Therefore we can start by immediately clearing the rest of the screen, which should only be the old trailer.
	tc.clearToEnd()
	// Behavior has to branch based on whether we're about to end up with a partial line on the end -- we have to save cursor before printing that if it's present, and buffer the partial line for some additional handling.
	// (... the cursor saving distinction has not turned out to be load bearing; see comments about r-A-J vs s/u approach -- but the buffers still are.)
	switch idxLastBr := bytes.LastIndexByte(msg, '\n'); idxLastBr {
	case -1: // If no breaks: entire thing is a fragment.  Cursor save unchanged.  Append buffer, write whole trailer (includes the partial buffer).
		tc.partial.Write(msg)
	case len(msg) - 1: // If break at the end: flush current partial and clear that buf, and print the entire current message.  Update saved cursor.  Now trailer.
		tc.partial.WriteTo(tc.wr)
		tc.wr.Write(msg)
		tc.cursorPositionSave()
	default: // If breaks in the middle: flush current partial, clear that buf, write the full lines, update cursor, then store the new trailing partial line.  Finally, trailer (includes the new partial buffer).
		tc.partial.WriteTo(tc.wr)
		tc.wr.Write(msg[0 : idxLastBr+1]) // +1 to include the linebreak.
		tc.cursorPositionSave()
		tc.partial.Write(msg[idxLastBr+1:]) // +1 to keep the linebreak out of the partial.
	}
	//fmt.Fprintf(os.Stderr, "(write %q; partial buf at end of write contains: %q)\n", msg, tc.partial.Bytes())
	tc.printTrailer()
	return len(msg), nil
}

func (tc *Controller) cursorPositionSave() {
	// tc.wr.Write([]byte("\x1B[s")) // Ask the terminal emulator itself to save the position.
}
func (tc *Controller) cursorPositionRestore() {
	// tc.wr.Write([]byte("\x1B[u")) // Ask the terminal emulator itself to return to the saved the position.
}
func (tc *Controller) cursorMoveUp(n int) {
	// fmt.Fprintf(os.Stderr, "(go up: %d)\n", n)
	// You'd think "0A" means "move up zero lines.  So actually don't do anything".  But terminals seem to take "0A" just as they would "1A" or plain "A".
	if n > 0 {
		tc.wr.Write([]byte("\r\x1B[" + strconv.Itoa(n) + "A"))
	}
}
func (tc *Controller) clearToEnd() {
	tc.wr.Write([]byte("\x1B[J")) // Clear from cursor to end of screen.
}

func (tc *Controller) currentTrailerHeight() int {
	h := len(tc.trailer)
	if tc.partial.Len() > 0 {
		return h + 1
	}
	return h
}

func (tc *Controller) printTrailer() {
	// Cursor should have already been returned to the checkpoint,
	// cleared from that point onward,
	// and any new content already printed.
	partial := tc.partial.Bytes()
	if len(partial) > 0 {
		tc.wr.Write(partial)
		tc.wr.Write([]byte{'\n'})
	}
	tc.wr.Write([]byte("\x1B[m")) // Styles from above should never leak into the trailer (nor into the next partial line emission, even if there is no trailer).
	for _, line := range tc.trailer {
		tc.wr.Write(line)
		tc.wr.Write([]byte{'\n'})
	}

	// Move the cursor back to the top of the trailer.
	// This makes the system more anti-fragile, because if any uncontrolled writes reach the terminal,
	//  now they'll smash over our trailer content... and while that may be a bummer, at least *we're* being a good actor:
	//   as long as that uncontrolled content ended with a linebreak, then we won't smash over it when we next rerender.
	//    It'll still end up flowing into scrollback, and overall the rendering should experience a graceful recovery.
	// There is one downer to doing this here, rather than just before repainting:
	//  it means the cursor is flashing somewhere above the trailer, which might not be what the user would visually expect.
	tc.cursorMoveUp(tc.currentTrailerHeight())
	tc.cursorPositionRestore()
}
