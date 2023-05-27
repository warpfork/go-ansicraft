go-ansicraft
============

A simple library for crafting ANSI control sequences for terminal rendering.


What can it do?
---------------

- Write and update text at the bottom of the screen, while also allowing regular scrollback to be emitted above that.
- That's about it.

(Simple is good.)

### What's the API?

You give us an `io.Writer` that represents the terminal.
We give you an object back that's another `io.Writer`, plus some superpowers.

If you call `SetTrailer` on that new writer, then the text you give it will stay sticky on the bottom of your output.
(Call `SetTrailer` repeatedly to animate!)

If you pass regular writes through it as an `io.Writer`, those just pass through and appear in regular scrollback.

### Where's the usage examples?

Check out the [demo directory](./demo/).

Use `go run` on the individual files there to see the demos in action.

### What can't it do?

It's not made for "fullscreen" terminal applications.  That's a whole different beast.

This library _does not put the terminal in raw mode_, which makes it a great deal simpler (and arguably more robust) than libraries which do that.

### What else doesn't it do?

Progress bars, fun stuff like that -- that's not bundled in here.

Other libraries _have_ got that part handled.

Hook them up by having them render to a buffer, and then call our `SetTrailer` function with the buffer content.
Then you can have your snazzy progress bars (or whatever), and also have plain scrollback flying by above.
Best of both worlds!

(If the library you're looking at for progress bars can't write to a buffer?  It's a bad library.)

### Any planned future work?

We might end up accumulating some more features like helpers for color and formatting.  _Maybe_.

If introduced, these will remain optional, and be oriented around composing byte slices -- simple, compositional, and nonmagical.



Why did this need another library?
----------------------------------

There are other terminal rendering assistance libraries.  This one is mine.

But more seriously, because that "updates at the bottom, scrollback above" thing is kinda tricky.
And ironically, it gets a lot *more* tricky if you try to make a library that hides the complexity of terminals from the user.
I found that feature was missing from every other library I looked at, and also impossible to compose with many of them, because they had _too much_ abstraction.

The main vibe I wanted from this one is truthful mechanical sympathy to the real system.
Help me interact with the terminal, but don't try to hide it, either.
Terminals are touchy beasts.
Trying to totally abstract yourself from the realities of them too much results in code you can't compose,
or has terrifying edge cases that you become powerless to handle if the library is trying to hard to shield you from messy reality.

So: we have *one* object that handles talking to terminals.
It wraps a regular `io.Writer`, and that's that.
And all of our other features are oriented around writing to `io.Writer` or buffers, and do *not* assume they have direct access to a TTY device...
because this is the closest thing to an actual composable API you can ever have with a terminal.

¯\_(ツ)_/¯



How robust is it?
-----------------

Extremely.  As much as I can make it.

The scrollback feature works with an extremely minimal number of escape codes.
It uses the codes that are the most widely and reliably supported by the broadest number of terminals and terminal emulators.

### What happens if I write partial lines?

The right thing happens.

- We'll buffer them;
- we'll write them out, and add a trailing line break (so the trailer can render cleanly, on a new line)
- and write the trailer.

When you keep appending the line, still without a linebreak?  Same thing, but we grow the buffer.
The whole line gets redrawn, correctly.

So if you want to write an elipsis with `write("."); sleep(1); write("."); sleep(1); write(".\n");` ... yeah, fine.
Do it.  It'll work.

### About feature detection...

The library actually does very little feature detection.
What feature detection this library does support, you must call upon it in order to use it.

It's my opinion that resisting feature-detection actually leads to *more* robustness in the current world:
every terminal emulates vt100 / handles ANSI control codes; admit it.
Attempting to perform feature-detection on terminal devices only leads to more platform-specific code, more edge cases to test, more errors to handle, and ultimately a worse experience.
In many cases, it even makes the end-user experience worse because you become more likely to experience feature-detection-gone-wrong than actual incompatbilities with the basic feature if you had just used them.

We do offer the very basic feature detection of "is this a terminal" and some norms for "did the user actively request 'dumb' mode";
if you want to detect this, there are functions that you can call.
(e.g.: We won't read environment variables... unless you ask for it.  Libraries shouldn't have global interactions.)

#### Not everybody emulates vt100!

Yes, they do.  They really do.

Or at the very least: close enough.

Are there edge cases in the CSI-s/u codes in various emulators?  Yes.  _So we don't use those._

Are there edge cases in the CSI-#-J and CSI-#-A codes in various emulators?  Nope.  Those are remarkably solid.  So we _do_ use those.

It's less scary than it sounds.

#### What about Windows?

No, seriously, even terminals on Windows emulate vt100, and use the same ANSI codes as everyone else.  This has been true for years.

There's absolutely no reason to ship platform specific code or add a dependency on cgo in order to support Windows.  There's just no need.

#### What about DUMB terminals?

You can ask for the detection for that.

If you don't, the library will assume a vt100.

(We don't know; maybe you *are* writing to a file and not a TTY, but you want the thing to cause colors and fun when you `cat` the file bare into a TTY later.  No judgement.  You should be able to do that if you want to do it.)

#### What if there are uncontrolled output streams?

See the next section.

It's as robust as we can make it... but there are limits.



How much does my application have to buy in?
--------------------------------------------

Unfortunately, if you want scrollback to work completely reliably, and the trailer to always remain painted at the bottom...
you do have to route all other writes through our `io.Writer` that wraps the terminal.
(You can't go straight to stdout and stderr anymore.)

I wish it weren't so; alas, 'tis.

This isn't a code quality thing; it's a reality-of-the-situation thing.
In order to make these features work, we have to control where the terminal cursor is at;
and we have to repaint the trailer any time any other content appears.
That means we need to do some work to maintain the state of the terminal every time there's any new output.

We _do_ make this as anti-fragile as possible, though.
We keep the cursor positioned such that any uncontrolled output will corrupt the trailer content,
but the next repaint of the trailer will _not_ paint over the uncontrolled output (as long as it ended in a linebreak).
This is as about antifragile as is possible within the contraints of terminal rendering.





License
-------

SPDX-License-Identifier: Apache-2.0 OR MIT
