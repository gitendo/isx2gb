### Notice
Starting October 12th, 2023 GitHub is enforcing mandatory [two-factor authentication](https://github.blog/2023-03-09-raising-the-bar-for-software-security-github-2fa-begins-march-13/) on my account.  
I'm not going to comply and move all my activity to GitLab instead.  
Any future updates / releases will be available at: [https://gitlab.com/gitendo/isx2gb](https://gitlab.com/gitendo/isx2gb)  
Thanks and see you there!
___

# isx2gb v1.03

Command line utility to convert Intelligent Systems eXecutable files into Game Boy ROM format and more. This is my personal replacement for abISX v1.02 by Anaerob. It has following improvements:

- code / data overflow in bank 0 is allowed, excess bytes are moved to bank 1 without any fuss.
- you're not forced to ORG your code / data if you don't need to, just GROUP it in appropriate banks.
- you can ORG your code to RAM or SRAM (why not?) and save it to file(s) for further processing. This might be appreciated by clever coders or when size does matter.
- you can patch Game Boy ROM on the fly exploiting ISX file as [IPS](http://justsolve.archiveteam.org/wiki/IPS_(binary_patch_format)) replacement. All the code / data will be applied to proper offsets.
- creating symbolic file is easy, just use CAPSOFF/SMALL with PUBALL directives in your source(s).
- ROM checksums are fixed automagically providing that there's a valid logo in header section.
- more readable ROM map.

### Options : 
```
⚠ there were changes from 1.02 to 1.03

-d  dump isx records into binary file(s)
-f  switch ROM filling pattern from 0x00 to 0xFF
-p  patch supplied ROM file with ISX records
-r  round up ROM size to the next highest power of 2
-s  create symbolic file for debugger
```

### Examples :
```
isx2gb -d code.isx
```
This will save all ORG sections into separate binary files, no ROM file is created. If you're into 128B/256B/1K/etc. coding, try to place your code in WRAM and compress such output for better result.
```
isx2gb -p patch.isx romfile.gbc
```
Patch `romfile.gbc` with code / data from `patch.isx` which basically works like `.ips` patching you're most likely familiar with. Take a look at my [Macross 7 english translation](https://github.com/gitendo/bm7j) for a proof of concept.
```
isx2gb -f -r -s game.isx
```
Create ROM file which will have free space filled with 0xFF instead of 0x00 (default). Round it to proper size, just like commercial stuff and produce symbolic file to be used with debugger ie. [BGB](http://bgb.bircd.org/).

### Important :
Let me know if you see similar message:
```
Error: Unknown record type (1417 : 14)
```
Currently only record types 0x01, 0x13, 0x14, 0x20, 0x21, 0x22 are supported and that should be enough.

### To do :
- improve rom padding option
- test it thoroughly

### Bugs :
Let me know if there's any.
