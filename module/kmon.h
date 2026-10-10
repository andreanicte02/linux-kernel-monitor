/* SPDX-License-Identifier: GPL-2.0 */
#ifndef KMON_H
#define KMON_H

#include <linux/seq_file.h>

/* Each show function renders one /proc/kmon_* file as JSON. */
int kmon_procs_show(struct seq_file *m, void *v);
int kmon_mem_show(struct seq_file *m, void *v);
int kmon_states_show(struct seq_file *m, void *v);

/* Writes s as a quoted JSON string, escaping anything that would break the document. */
void kmon_json_string(struct seq_file *m, const char *s);

#endif
