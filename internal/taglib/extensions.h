#ifndef EXTENSIONS_H
#define EXTENSIONS_H

#ifdef __cplusplus
extern "C" {
#endif

#include <taglib/tag_c.h>

void taglib_set_item_mp4(TagLib_File *file, const char *key, const char *value);
void taglib_set_picture(TagLib_File *file, const char *data, unsigned int size, const char *desc, const char *mime, const char *typ);

#ifdef __cplusplus
}
#endif

#endif // EXTENSIONS_H