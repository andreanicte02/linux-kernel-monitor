// SPDX-License-Identifier: GPL-2.0
#include "kmon.h"

void kmon_json_string(struct seq_file *m, const char *s)
{
	seq_putc(m, '"');
	for (; *s; s++) {
		unsigned char c = *s;

		if (c == '"' || c == '\\') {
			seq_putc(m, '\\');
			seq_putc(m, c);
		} else if (c < 0x20 || c >= 0x7f) {
			/* Control bytes and non-ASCII: escape so the output is always valid JSON. */
			seq_printf(m, "\\u%04x", c);
		} else {
			seq_putc(m, c);
		}
	}
	seq_putc(m, '"');
}
