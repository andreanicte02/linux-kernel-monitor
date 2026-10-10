// SPDX-License-Identifier: GPL-2.0
#include <linux/rcupdate.h>
#include <linux/sched.h>
#include <linux/sched/signal.h>

#include "kmon.h"

/*
 * Counts processes by the same one-letter state ps(1) shows. Every process is
 * visited exactly once: for_each_process already includes children, so they
 * are not walked again.
 */
int kmon_states_show(struct seq_file *m, void *v)
{
	unsigned long running = 0, sleeping = 0, disk_sleep = 0, stopped = 0;
	unsigned long zombie = 0, idle = 0, other = 0, total = 0;
	struct task_struct *p;

	rcu_read_lock();
	for_each_process(p) {
		switch (task_state_to_char(p)) {
		case 'R': running++; break;
		case 'S': sleeping++; break;
		case 'D': disk_sleep++; break;
		case 'T':
		case 't': stopped++; break;
		case 'Z': zombie++; break;
		case 'I': idle++; break;
		default: other++; break;
		}
		total++;
	}
	rcu_read_unlock();

	seq_printf(m,
		   "{\"running\":%lu,\"sleeping\":%lu,\"disk_sleep\":%lu,\"stopped\":%lu,"
		   "\"zombie\":%lu,\"idle\":%lu,\"other\":%lu,\"total\":%lu}\n",
		   running, sleeping, disk_sleep, stopped, zombie, idle, other, total);
	return 0;
}
