# startup-time-checker

A self-imposed lockout: pick a time, and the machine will not finish booting
before it. Handy for "no more Factorio tonight" — shut down, and the computer
stays unusable until morning.

## Usage

```sh
startup-time set "tomorrow 07:00"   # any format date(1) understands
startup-time get                    # show the current lock
startup-time clear                  # remove it
```

`startup-time` is setuid root, so no sudo is needed. Set the time, and once
you are happy with it, shut down: the machine stays locked until then.

On boot, `startup-time-checker.service` runs before `sysinit.target` and waits
(showing the time left under the boot splash logo, and on the console) until the lock has passed. A missing
or corrupt state file never blocks booting.

### Making it hard to cheat

Keep the hardware clock in UTC and set it from the system clock, so the BIOS
clock is the only way to move time:

```sh
sudo timedatectl set-local-rtc 0
sudo hwclock --systohc --utc
```

Setting a BIOS/UEFI password closes that gap too.

### Emergency escape

At the GRUB menu press `e`, add `systemd.mask=startup-time-checker.service`
to the `linux` line and boot with Ctrl+X. Then `startup-time clear`.

## Install

Download the `.deb` from the latest
[release](https://github.com/ukarlsson/startup-time-checker/releases) and:

```sh
sudo apt install ./startup-time-checker_*_amd64.deb
```

## Build

```sh
sudo apt install build-essential debhelper golang-go
dpkg-buildpackage -us -uc -b      # .deb ends up in ../
```

Tests (`go test ./...`) run as part of the build. GitHub Actions builds the
package in an `ubuntu:26.10` container on every push; pushing a `v*` tag
publishes a release with the `.deb` attached.
