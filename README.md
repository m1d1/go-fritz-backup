# go-fritz-backup

A backup utility for AVM FRITZ!Box® devices using the TR-064 protocol.

> ⚠️ This project is not affiliated with or endorsed by AVM GmbH.  
> **FRITZ!Box®** is a registered trademark of AVM Computersysteme Vertriebs GmbH, Berlin (Germany).

## Features

- Backup of configuration file
- Backup of phonebooks
- Backup of call barring list (caller blocklist)
- Backup of phone assets (e.g., ringtones)
- Sample automation scripts (`backup-fritz.sh`, `backup-fritz.ps1`) with backup rotation

## Project status

This project is tested and works on a FRITZ!Box 7490 with FRITZ!OS 7.60.  
It is not yet tested with:

- Other FRITZ!Box models
- TOTP / 2FA enabled in any way.
- Non-default user accounts

If you encounter issues, first verify TR-064 is enabled (see **requirements**)  
The `phone_assets` export is special, disable it if it isn't working.  
This is the only feature not using TR-064.

> TR-064 does not support TOTP or any kind of 2FA by design.  
> I briefly did a test using two separate user accounts on my old Fritz!Box 7490.  
> As soon as TOTP was enabled for only one of the accounts, **go-fritz-backup** didn't work  
> for that other normal account anymore.  
> TR-064 returned empty uris - you may see errors like `ERROR parse "://": missing protocol scheme`  
> Good luck!

## Requirements

- A FRITZ!Box with enabled TR-064 access. Turn the checkbox on:  
  `Home Network > Network > Network Settings > Additional Settings > Allow access for applications`  
  If you can open `http://<your-box-ip>:49000/tr64desc.xml` in a browser - it's enabled.

- A FRITZ!Box device user account with **disabled TOTP!!!**  
  To create a new user (or to get the existing default username) go to:  
  `System > FRITZ!Box Users`. Click `Add User` to add a new account.

- A configured `backup-config.yml` file in the same directory as the binary.

## Configuration

Copy `backup-config.yml.example` to `backup-config.yml` and adjust it to fit your environment.  
Use the username from `System > FRITZ!Box Users`. Or create a new user.

### backup-config.yml

⚠️ Do **not** use tabs in the YAML file. Only use spaces.

- **device:**  
  Contains the FRITZ!Box username, password, and host.  
  Ensure the user account has sufficient permissions.

- **export:**

  - The configuration file will **always** be backed up.  
    The `export` > `password` is mandatory for the export and must **not be empty**.  
    It's needed for restoring the backup (`System > Backup > Restore`).

  - `phone_books`: if `true`, all phonebooks will be exported to separate `.xml` files. `false` to turn it off.

  - `phone_barringlist`: if `true`, the caller blocklist will be exported. `false` to turn it off.

  - `phone_assets`: `true` to export custome ringtones etc.  
     this is the only export that does **not** use TR-064.  
     If it fails - set it to `false`.

- **backup:**
  - `target_path`: Target directory where backup files will be stored.  
    Make sure it exists and is writable by the user running the binary.  
    For Windows: double-backslash all path separators (e.g., `C:\\Backup`)

### Linux (systemd timer / aka cronjob)

Create `/etc/systemd/system/backup-fritz.service`:

```ini
[Unit]
Description=Backup FRITZ!Box daily

[Service]
Type=simple
Nice=19
IOSchedulingClass=2
IOSchedulingPriority=7
ExecStart=/home/youruser/backup-fritz.sh
#User=youruser
#Group=yourgroup
```

Create corresponding `/etc/systemd/system/backup-fritz.timer`  
Adjust interval of backup in `OnCalendar` (Set to daily 23:52 / 11:52pm)

```ini
[Unit]
Description=Backup FritzBox daily

[Timer]
WakeSystem=false
OnCalendar=*-*-* 23:52:00
#RandomizedDelaySec=10min
Persistent=true

[Install]
WantedBy=timers.target
```

Enable and start the timer once:

```
systemctl daemon-reload
systemctl enable backup-fritz.timer
systemctl start backup-fritz.timer
```

Check active timers with
`systemctl list-timers`

### Windows

Create Windows Task Manger Task, running `backup-fritz.ps1` (...)

## MIT License

Copyright © 2025 Michael Dinkelaker

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the “Software”), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

## Disclaimer

This software is provided "as is" without any warranty.
AVM GmbH is not associated with or responsible for this project.
Use at your own risk.
