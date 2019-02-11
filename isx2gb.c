#include <ctype.h>
#include <stdio.h>
#include <string.h>
#include <stdlib.h>

#include "isx2gb.h"

FILE *fp = NULL;
void *isxp = NULL;

void banner(void)
{
	printf("\nisx2gb v%.2f - ISX to Game Boy ROM converter\n", VERSION);
	printf("Programmed by: tmk, email: tmk@tuta.io\n");
	printf("Project page: https://github.com/gitendo/isx2gb/\n\n");
}


void usage(void)
{
	banner();

	printf("Syntax: isx2gb.exe [options] file.isx\n\n");
	printf("Options:\n");
	
	printf("\t- \n");

	exit(EXIT_FAILURE);
}


void handler(char error)
{
	char	*msg;

	switch(error)
	{
		case ERR_UNK_OPTION :	msg = "Invalid option!\n"; break;
		case ERR_IF_OPEN	:	msg = "Input file not found or access denied!\n"; break;
		case ERR_OF_OPEN	:	msg = "Invalid output file name or access denied!\n"; break;
		default				:	msg = "Undefined error code!\n"; break;
	}

	if(fp)
		fclose(fp);
	if(isxp)
		free(isxp);

	printf("\nError: %s", msg);

	exit(EXIT_FAILURE);
}


int main (int argc, char *argv[])
{
	char				arg, *ext = NULL, fname[FILENAME_MAX], options, rt, status;
	unsigned char		bank[16384], header[32];
	int					*sig;
	long				fsize;

	if(argc < 2)
		usage();

	for(arg = 1; arg < argc; arg++)
	{
		if((argv[arg][0] == '-') || (argv[arg][0] == '/'))
		{
			switch(tolower(argv[arg][1]))
			{
//				case 'c': options |= FLAG_; break;

//				default: handler(ERR_UNK_OPTION); break;
			}
		}
		else
		{
			ext = strstr(argv[arg], ".isx");
			if(ext)
				strcpy(fname, argv[arg]);
		}
	}

	fp = fopen(fname, "rb");
	if(fp == NULL)
		handler(ERR_IF_OPEN);
	fseek(fp, 0L, SEEK_END);
	fsize = ftell(fp);
	if(fsize <= ISX_HEADER)
		handler();
	rewind(fp);
	isxp =  malloc(fsize);
	if(isxp == NULL)
		handler();
	fread(isxp, fsize, 1, fp);
	fclose(fp);

	sig = (int *) isxp;
	if(*sig != ISX_SIGNATURE)
		handler();

	ext = strstr(fname, ".isx");
	strcpy(ext, ".gb");
	fp = fopen(fname, "wb");
	if(fp == NULL)
		handler(ERR_OF_OPEN);

	printf("%.32s\n", (char *) isxp);
	isxp += ISX_HEADER;
	memset(bank, 0, sizeof(bank));

	while(fsize > 0)
	{
		rt = getc(fp);
		fsize--;

		switch(rt)
		{
			case 0x01:
				bnk = getc(fp);
				fsize--;
				break;
			case 0x03:
				break;
			case 0x04:
				break;
			case 0x11:
				break;
			case 0x13:
				break;
			case 0x14:
				break;
			default:
				printf("unknown record type\n");
				handler();
		}
	}

	free(isxp);
	fclose(fp);
}
