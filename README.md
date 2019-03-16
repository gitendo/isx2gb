# isx2gb v1.00

Command line utility to convert Intelligent Systems eXecutable files into Game Boy ROM format. This is my personal replacement to abISX v1.02 by Anaerob. It has following features:

- code / data overflow in bank 0 is allowed, excess bytes are moved to bank 1 without any fuss.
- ROM checksums are fixed automagically providing that there's a valid logo in header section.
- you can originate your code in RAM or SRAM (why not?) and use `-r` option to save it to file for further processing. This might also be appreciated by ROM hackers while generating series of patches.
- ROM map is more readable.

### Options :
```
-f  switch ROM filling pattern to 0xFF
-p  round up ROM size to the next highest power of 2
-r  save isx records separately
```

### To do :
- add option to create symbol files
- clean up code / comments
- test it thoroughly

### Bugs :
Hopefully none. Let me know if you find any.