#include <ctype.h>
#include <stdio.h>
#include <string.h>
#include <stdlib.h>

#include "isx2gb.h"

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


void error_handler(FILE *ifp, FILE *ofp)
{
	if(ifp)
		fclose(ifp);
	if(ofp)
		fclose(ofp);
	exit(EXIT_FAILURE);
}


int main (int argc, char *argv[])
{
	char				arg, *ext = NULL, fname[FILENAME_MAX], options, status;
	unsigned char		bank[16384], header[32];
	int					*sig;
	long				fsize;
	FILE				*ifp = NULL, *ofp = NULL;

	if(argc < 2)
		usage();

	for(arg = 1; arg < argc; arg++)
	{
		if((argv[arg][0] == '-') || (argv[arg][0] == '/'))
		{
			switch(tolower(argv[arg][1]))
			{
//				case 'c': options |= FLAG_; break;

//				default: error_handler(ERR_UNK_OPTION); break;
			}
		}
		else
		{
			if(ifp == NULL)
			{
				ext = strstr(argv[arg], ".isx");
				if(ext)
				{
					ifp = fopen(argv[arg], "rb");
					fseek(ifp, 0L, SEEK_END);
					fsize = ftell(ifp);
					rewind(ifp);
					strcpy(ext, ".gb");
					ofp = fopen(argv[arg], "wb");
				}
			}
		}
	}

	if(ifp == NULL || fsize <= 32 || ofp == NULL)
		error_handler(ifp, ofp);

	fread(header, sizeof(header), 1, ifp);
	sig = (int *) header;

	if(*sig != 0x20585349)
		error_handler(ifp, ofp);


	printf("%.32s\n", header);
	memset(bank, 0, sizeof(bank));

	fclose(ifp);
	fclose(ofp);
}
