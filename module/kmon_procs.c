// SPDX-License-Identifier: GPL-2.0
#include <linux/cred.h>
#include <linux/mm.h>
#include <linux/rcupdate.h>
#include <linux/sched.h>
#include <linux/sched/mm.h>
#include <linux/sched/signal.h>
#include <linux/user_namespace.h>

#include "kmon.h"

/* Resident memory in KiB, or 0 for kernel threads (which have no mm). */
static unsigned long kmon_rss_kb(struct task_struct *p)
{
	unsigned long rss = 0;
	struct mm_struct *mm;

	/* task_lock keeps p->mm from being torn down while we read its counters. */
	task_lock(p);
	mm = p->mm;
	if (mm)
		rss = get_mm_rss(mm) << (PAGE_SHIFT - 10);
	task_unlock(p);
	return rss;
}

/*
 * One JSON object per process (thread-group leader). The parent PID is
 * included instead of nesting children, so user space can rebuild the full
 * tree at any depth.
 */
int kmon_procs_show(struct seq_file *m, void *v)
{
	struct task_struct *p;
	bool first = true;

	seq_puts(m, "[");
	rcu_read_lock();
	for_each_process(p) {
		char comm[TASK_COMM_LEN];

		get_task_comm(comm, p);
		seq_printf(m, "%s\n{\"pid\":%d,\"ppid\":%d,\"name\":",
			   first ? "" : ",", task_tgid_nr(p),
			   task_tgid_nr(rcu_dereference(p->real_parent)));
		kmon_json_string(m, comm);
		seq_printf(m, ",\"state\":\"%c\",\"uid\":%u,\"threads\":%d,\"rss_kb\":%lu}",
			   task_state_to_char(p),
			   from_kuid_munged(&init_user_ns, task_uid(p)),
			   get_nr_threads(p), kmon_rss_kb(p));
		first = false;
	}
	rcu_read_unlock();
	seq_puts(m, "\n]\n");
	return 0;
}
