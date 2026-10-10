// SPDX-License-Identifier: GPL-2.0
#include <linux/mm.h>

#include "kmon.h"

int kmon_mem_show(struct seq_file *m, void *v)
{
	struct sysinfo si;
	unsigned long page_kb = PAGE_SIZE / 1024;

	/* sysinfo counts pages of si.mem_unit bytes, not bytes. */
	si_meminfo(&si);
	seq_printf(m,
		   "{\"total_kb\":%lu,\"free_kb\":%lu,\"available_kb\":%lu,"
		   "\"buffers_kb\":%lu,\"shared_kb\":%lu}\n",
		   si.totalram * si.mem_unit / 1024,
		   si.freeram * si.mem_unit / 1024,
		   (unsigned long)si_mem_available() * page_kb,
		   si.bufferram * si.mem_unit / 1024,
		   si.sharedram * si.mem_unit / 1024);
	return 0;
}
