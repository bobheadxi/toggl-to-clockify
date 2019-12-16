# toggle-to-clockify

A simple exporter for syncing a project on [Toggl](https://toggl.com/) to [Clockify](clockify.me/).
I use this to migrate work-related hours from my personal time tracker on Toggl to
my employer's time tracker on Clockify.

### Usage

Use `toggl-to-clockify -help` to display help text. Example usage:

```sh
$> toggl-to-clockify \
    -start=16 \
    -toggl.project="sumus-portal" \
    -clockify.project="Sumus Portal" \
    -clockify.workspace="Andrew's workspace" \
    -clockify.billable
```

Add the `-exec` flag to actually run the update once you've checked the entries being
created are correct.
