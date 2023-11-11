# disco

```
+----------------------------------------------------------------------+
|           ===========   =====  ==========   ========     ========    |
|           ===     ===   ===  ===      ==  ===    ===   ===    ===    |
|          ===      ===  ===  ====        ===          ===      ===    |
|         ===      ===  ===   ====       ===          ===      ===     |
|        ===      ===  ===     =====    ===          ===      ===      |
|       ===      ===  ===        ====  ===          ===      ===       |
|      ===      ===  ===         ==== ===          ===      ===        |
|     ===     ===   ===  ==      ===  ===    ===   ===    ===          |
|   ===========   ===== ==========    ========     ========            |
+----------------------------------------------------------------------+
|            Domestic Illumination System Control Operator             |
+----------------------------------------------------------------------+
```


## about

DISCO is a system for controlling home lighting. It's a text protocol
that can talk to multiple backends
([Phillips Hue](https://developers.meethue.com/) and
[LIFX LAN](https://lan.developer.lifx.com/docs) at this stage).

A command line tool (`disco`) and a web server (`discod`) lie within.

Both `disco` and `discod` are configured by a single yaml file (`disco.yml`).

**These tools are absolutely not production ready.** They may never be. I wrote
this for a pet project and I'm sharing it because I think it's fun.


## dedication

This repository contains some of the best and the worst code I have ever
written. It is dedicated to my mentor, who helped me grow from a person who
can code to an engineer. We didn't get enough time together. May your
goroutines never deadlock in heaven my friend.


## attribution

The [XKCD Color Survey](https://xkcd.com/color/rgb/) results were used to
create `color/xkcd.go` and are licensed under
[CC0 1.0 Universal](https://creativecommons.org/publicdomain/zero/1.0/).

The Noto Sans Mono font served by the web ui belongs to the
[Noto Project Authors](https://github.com/notofonts) and is provided under the
[SIL Open Font License](http://scripts.sil.org/OFL).

The web ui favicon was generated using [favicon.io](https://favicon.io). The
font used is Leckerli One by [Gesine Todt](www.gesine-todt.de) also under the
[SIL Open Font License](http://scripts.sil.org/OFL).


## package layout

```
.
├── bin                # build and deploy scripts
├── cmd
│   ├── color          # playground for the color package, may or may not build
│   ├── disco          # command line tool
│   ├── discod         # web server
│   ├── hue            # playground for the hue package, may or may not build
│   └── lifx           # playground for the lifx package, may or may not build
├── color              # color conversion utilities
├── disco.example.yml  # example configuration file
├── disco.go           # core text protocol
├── hue                # thin wrapper over hue api
├── huecmd             # translation layer between text protocol and hue
├── lifx               # thin-ish? lifx lan client
└── lifxcmd            # translation layer between text protocol and lifx
```

## text protocol

In the beginning is the command.

```go
type Cmd struct {
	Action string
	Target string
	Args   []string
}
```

A command comprises an `Action` (what I want to do), a `Target` (what I want
to do it to), and one or more `Args` or parameters (what am I doing).


### switch, dim, color

At first, there are three actions:
- `switch`
- `dim`
- `color`

Meaning, I want to switch a device on or off (`switch light1 on`), I want
to dim the brightness to a certain level `[0,100]` (`dim light1 50`), or I
want to set the color to an RGB hex value (`color light1 ff0000`). Just for
fun, the color command understands the xkcd color survey names so I can
`color light1 raspberry`.

There is some nuance to color setting, as we are working with lamps not
monitors. Under the hood, the brightness values are stripped out and colors
are always set as though they were maximum brightness. I find RGB values
much easier to comprehend than HSV or XY so we stick to that space and accept
that some colors in the low brightness range will not appear as expected.

This feels a little strange at first but it's workable. The result is we have
decoupled color from brightness (dimming) so the `dim` and `color` commands are
orthogonal.

### get state

A command with no args is a getter. It is asking what the args would be for that
action and target. The response takes the form of a command.

```sh
> disco switch light1
switch light1 on
```

The target can be omitted to get everything.

```sh
> disco switch
switch light1 on
switch light2 off
switch light3 off
...
```

#### timing

The `dim` and `color` commands support a second arg which is a time duration.
Example `dim light1 50 6s`. The default duration is `3s`, which any lighting
tech will tell you is a standard fade.

I tried setting durations on the switch command and it didn't end well. So the
switch is instantaneous. Which is fine, that is how we expect a switch to
behave.


#### decomposition of targets

The backends decompose devices into zero or more targets applicable to each
of the three actions. So a smart plug that has no dimming or color capability
will only be addressable by the switch command. A hue gradient lightstrip
with one switch, one dimmer, and 5 color zones presents those
accordingly.

I have made some sacrifices to the flexibility of controlling some devices
to suit simplicity and I am happy the system does everything I want it to do.


### prefix, map, link

The hue and lifx packages refer to devices by ID. The hue and lifx backends
have their own friendly names for lights and that's nice for them but I
didn't feel like plumbing all that through. In the interest of simplicity,
all of DISCO's configuration is completely static in `disco.yml`.

To the backends, the target is a device ID.

IDs from the backends are prefixed into namespaces for clarity and to avoid
collisions if two backends happened to use the same ID scheme in future.

DISCO will represent a hue device with id `ae5cdf75-fe52-4f1c-8e6e-cb4ad3786085`
with a prefix like this `hue/ae5cdf75-fe52-4f1c-8e6e-cb4ad3786085`.

Which brings us to the `Map`, a simple text map that allows us to give
friendly names to devices.

```yaml
Map:
  hue/ae5cdf75-fe52-4f1c-8e6e-cb4ad3786085: light1
```

With this in the config and we can refer to the device as `light1`. Easy.

The `Link` is for grouping devices together.

```yaml
Link:
  lights: [light1, light2, light3]
  more: [light4, light5, light6]
  all: [lights, more]
```

When a link target is expanded, the original command is rewritten into one
command for each target that is linked.

For example, `switch lights on` becomes

```
switch light1 on
switch light2 on
switch light3 on
```

Links can link links. The implementation for this is iterative and loops over
commands until all links are resolved.

**Footgun** there is no detection for circular links so watch out. You have
been warned.


### cue

A `Cue` is slice of command with a slug and a friendly name for the Web UI.

```yaml
Cue:
  light-on:
    Text: Light On
    Cmds:
      - switch all on
```

Cue is also a command action, with this config in place we can run
`disco cue light-on`. Cues can cue cues.

**Footgun** there is no detection for a circular cue reference. Please, do not.


### chase

Now we're getting serious. A `Chase` is a slice of slices of command. Each of
which should include a `wait` before the next step. The `wait` action is only
applicable in a chase. Chases loop forever until they are stopped.

```yaml
Chase:
  all-soft:
    Text: All Soft
    Steps:
      -
        - dim all 100 6s
        - wait 7s
      -
        - dim all 80 6s
        - wait 7s
```

Chases only run in `discod`, for now. A chase is not a command action and
chases _cannot_ chase chases. Can you imagine.


### sheet

The cue sheet is the last piece of configuration and describes the web page
displayed by `discod`. The interface comprises groups of buttons divided
into sections. That is all. A button may call a `Cue` or a `Chase`.

For example

```yaml
Sheet:
  - Text: Main
    Group:
      -
        - Cue: all-on
  - Text: Chase
    Group:
      -
        - Chase: all-soft
```


## discod

The web server is very simple. It renders an html page of buttons according
to the configuration of the `Sheet`. Following the sheet is an unordered
list of running chases with a button to stop them.

Cues are sent to the `/cue/{name}` endpoint and handled by the cue handler,
which returns a 204 No Content.

Chases are sent to the `/chase/{name}` endpoint and handled by the chase
handler which returns a 302 Found to `/`. The `/chase/{name}/stop` endpoint
will stop a chase.

The webserver is fully self-contained, no frameworks, no javascript, serves
its own font and has a manifest allowing it to be added to the home screen
as a web app.

`/bin/discod.sh` is an example deploy script over ssh to a server which makes
some assumptions (running discod as a runit service and non privileged user).

Running a deploy stops the service momentarily and kills any running chases.
If you need zero downtime in your home setup I'm sure you can figure
something out...


## design principles

Simple. Expressive. Robust. Effective.

The whole system runs without ever calling out to the internet. No internet
outage will stop the DISCO.

Highly opinionated. Built for engineers. Easy to operate. The only things
the web interface can do must be codified into the `disco.yml`. In practice
this works just fine. You don't need a slider to dim the lights. Just pick
a few sensible settings and expose them.

No state. Mostly no state. The only state we hold in memory are the running
chases.

Good neighbor. The system can be used in conjunction with the hue and lifx
apps without any ill effects.


## what's missing

Oh so many things.

### validation of literally anything

There is some assumption if you made it this far you know what you're doing.

### tests

Test coverage of the color conversion utilities is decent. That's hardcore
stuff and it needs to be rock solid. Tests for everything else... let's call
that a later problem.

### cue timing

It might be nice to accept a time duration for the `cue` command and have it
override the timing of the commands within.

### hue bridge discovery

Just give it a static IP and a local DNS name.

### hue bridge registration

The hue docs explain how to do this. If I were to implement it it would likely
go into a command line tool.

### hue secret key

The hue api key should be handled as a secret and not a field in `disco.yml`.

### hue https

We're rolling `InsecureSkipVerify` for now. Phillip's https api is not fully
rolled out and my older bridge uses a self signed cert and this all seems like
a later problem.

### hue and lifx device registration

The manufacturer's apps are fine for this.

### device support

At the moment, DISCO supports hue plug, hue color bulbs, hue lightstrip and
gradient lightstrip, lifx color bulbs. Extending support to other devices
is certainly possible, but I don't own any (hint hint, anyone from Phillips
or LIFX reading this :wink:).

There is no support for white/color temperature at this stage. RGB color
values are packed into a uint32 so we could potentially use the last 8 bits
for color temp.
