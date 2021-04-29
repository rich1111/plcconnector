#include "bridge.h"

void bridge_int_func(intFunc f, int a, int b, char *c, int d, int e, void *g)
{
	f(a, b, c, d, e, g);
}
