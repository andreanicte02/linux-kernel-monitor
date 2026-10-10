// SPDX-License-Identifier: GPL-2.0
/*
 * kmon: exposes the process list, memory usage and process-state counts
 * as JSON files under /proc, for the kmon API and dashboard to read.
 */
#include <linux/init.h>
#include <linux/kernel.h>
#include <linux/module.h>
#include <linux/proc_fs.h>

#include "kmon.h"

struct kmon_entry {
	const char *name;
	int (*show)(struct seq_file *m, void *v);
};

static const struct kmon_entry kmon_entries[] = {
	{ "kmon_procs", kmon_procs_show },
	{ "kmon_mem", kmon_mem_show },
	{ "kmon_states", kmon_states_show },
};

static void kmon_remove_entries(size_t count)
{
	while (count--)
		remove_proc_entry(kmon_entries[count].name, NULL);
}

static int __init kmon_init(void)
{
	size_t i;

	for (i = 0; i < ARRAY_SIZE(kmon_entries); i++) {
		/*
		 * proc_create_single() hides the file_operations/proc_ops split
		 * introduced in 5.6, so the same code builds on old and new kernels.
		 */
		if (!proc_create_single(kmon_entries[i].name, 0444, NULL,
					kmon_entries[i].show)) {
			pr_err("kmon: could not create /proc/%s\n", kmon_entries[i].name);
			kmon_remove_entries(i);
			return -ENOMEM;
		}
	}
	pr_info("kmon: loaded, reading from /proc/kmon_{procs,mem,states}\n");
	return 0;
}

static void __exit kmon_exit(void)
{
	kmon_remove_entries(ARRAY_SIZE(kmon_entries));
	pr_info("kmon: unloaded\n");
}

module_init(kmon_init);
module_exit(kmon_exit);

MODULE_LICENSE("GPL");
MODULE_AUTHOR("Tikiram Samaneb Ruiz Manuel");
MODULE_DESCRIPTION("Process, memory and process-state monitor exposed as JSON in /proc");
