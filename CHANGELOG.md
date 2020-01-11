## 0.4.2 / 2020-01-11

* [BUGFIX] Fix flags with defaults that aren't required anymore [#96](https://github.com/metalmatze/alertmanager-bot/pull/96).

## 0.4.1 / 2020-01-07

Update to Go 1.13 and dependencies.

* [BUGFIX] Fix default template path [#55](https://github.com/metalmatze/alertmanager-bot/pull/55).
* [BUGFIX] Bring back defaults for flags [#93](https://github.com/metalmatze/alertmanager-bot/pull/93).

## 0.4.0 / 2019-02-19

* [FEATURE] Add ability to use templates for Telegram messages [#32](https://github.com/metalmatze/alertmanager-bot/pull/32).
* [ENHANCEMENT] Truncate too large messages [#52](https://github.com/metalmatze/alertmanager-bot/pull/52), thanks @BulatSaif.

## 0.3.1 / 2018-06-11

* [BUGFIX] Escape emojis in messages [#22](https://github.com/metalmatze/alertmanager-bot/pull/22), thanks @caarlos0.

## 0.3.0 / 2018-05-15

* [FEATURE] Allow for multiple bot admin users [#19](https://github.com/metalmatze/alertmanager-bot/pull/19), thanks @slrz.
* [FEATURE] Add log.level and log.json as cli flags [[9050ff4]](https://github.com/metalmatze/alertmanager-bot/commit/9050ff418bf5a07fcd684fb01fa7838a36b0af38).
* [ENHANCEMENT] Handle Ctrl+C interrupts and shutdown bot gracefully [[07c2356]](https://github.com/metalmatze/alertmanager-bot/commit/07c23563800e62e97cc0437a47cefd1aea332a82).
* [ENHANCEMENT] Internal refactoring of HTTP calls [#15](https://github.com/metalmatze/alertmanager-bot/pull/15), thanks @vtolstov.

## 0.2.1 / 2017-11-04

* [BUGFIX] /chats command send to the sender and not the chat [#10](https://github.com/metalmatze/alertmanager-bot/issues/10)

## 0.2.0 / 2017-10-17

* [BREAKING] Change the `STORE` env var to switch on the store backend instead of path
* [FEATURE] The Bot now ignores its own name as suffix on command in a group chat [[aae2dc5]](https://github.com/metalmatze/alertmanager-bot/commit/aae2dc5c1dae5f865cd697cb649fb757b7efaa6f) 
* [FEATURE] Use libkv as store backend [[6419064]](https://github.com/metalmatze/alertmanager-bot/commit/64190646e71910a10fdcb6e7533dc8a8dc485fec)
* [ENHANCEMENT] Use [dep](https://github.com/golang/dep) to pin dependencies [[b7e085b]](https://github.com/metalmatze/alertmanager-bot/commit/4bfd3f7a2ec559eee712f37ba8e32ca848717905)
* [ENHANCEMENT] Use the Option pattern to change the Bot's default values in `NewBot()`.

## 0.1.0 / 2017-03-31

Initial release.
