#ifndef ISX2GB_H
#define ISX2GB_H

#define VERSION 		1.0
#define ISX_HEADER		32
#define ISX_SIGNATURE	0x20585349

enum {  NO_ERROR,
		ERR_UNK_OPTION,
		ERR_NOT_FOUND
};

#pragma pack(push, 1)

typedef struct _RT01H {
	char	bnk;
	short	adr;
	short	len;
} RT01H;

#pragma pack(pop)



#endif /* ISX2GB_H */
