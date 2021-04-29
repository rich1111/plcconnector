#include "stdint.h"
#include "stdio.h"
#include "stdlib.h"
#include "unistd.h"

#include "libplcconnector/libplcconnector.h"

void callback(int service, int status, char *name, int type, int count, void *data)
{
    int i = 0;

    switch (service) {
    case Reset:
        printf("Reset\n");
        // here goes Reset service
        break;
    case ReadTag:
        printf("Read Tag\n");
        break;
    case WriteTag:
        printf("Write Tag\n");
        break;
    default:
        printf("unknown service\n");
        break;
    }
    switch (status) {
    case Success:
        printf("Succes\n");
        break;
    case PathSegmentError:
        printf("PathSegmentError\n");
        break;
    default:
        printf("unknown status\n");
        break;
    }

    // reading data (ReadTag - data read, WriteTag - data written)
    if ((service == ReadTag || service == WriteTag) && status == Success && name != NULL && data != NULL) {
        printf("%s %d\n", name, count);
        switch (type) {
        case TypeBOOL:
            printf("BOOL type [ ");
            for (uint8_t *d = (uint8_t *) data; i < count; i++) {
                printf("%hhd ", d[i]);
            }
            break;
        case TypeSINT:
            printf("SINT type [ ");
            for (int8_t *d = (int8_t *) data; i < count; i++) {
                printf("%hhd ", d[i]);
            }
            break;
        case TypeINT:
            printf("INT type [ ");
            for (int16_t *d = (int16_t *) data; i < count; i++) {
                printf("%hd ", d[i]);
            }
            break;
        case TypeDINT:
            printf("DINT type [ ");
            for (int32_t *d = (int32_t *) data; i < count; i++) {
                printf("%d ", d[i]);
            }
            break;
        case TypeREAL:
            printf("REAL type [ ");
            for (float *d = (float *) data; i < count; i++) {
                printf("%e ", d[i]);
            }
            break;
        case TypeDWORD:
            printf("DWORD type [ ");
            for (int32_t *d = (int32_t *) data; i < count; i++) {
                printf("%d ", d[i]);
            }
            break;
        case TypeLINT:
            printf("LINT type [ ");
            for (int64_t *d = (int64_t *) data; i < count; i++) {
                printf("%ld ", d[i]);
            }
            break;
        default:
            printf("unknown type [ ");
            break;
        }
        printf("]\n");
    }
    printf("\n");

    // name and data have to be freed
    free(name);
    free(data);
}

int main(void)
{
    // initialization
    plcconnector_init();

    // initialization of TABLE_DINT_1, type DINT, length 100
    plcconnector_add_tag("TABLE_DINT_1", TypeDINT, 100);

    // do not show debugging information (1 - show)
    plcconnector_set_verbose(0);

    // function called when data from PLC arrive
    plcconnector_callback(callback);

    // WWW page (listening address, port)
    plcconnector_serve_http("0.0.0.0", 28080);

    // listening address, port
    plcconnector_serve("0.0.0.0", 10000);

    int32_t data[3] = {0, 1, 2};

    while (1) {
        // update of TABLE_DINT_1
        data[0]++;
        data[1] += 2;
        data[2] += 3;
        plcconnector_update_tag("TABLE_DINT_1", 0, (void *) data, sizeof data);
        // update of TABLICA_DINT_1 with another offset
        plcconnector_update_tag("TABLE_DINT_1", 50, (void *) data, sizeof data);

        // here goes other code
        usleep(1000000);

        // terminate example after 60 seconds
        printf("%d\n", data[0]);
        if (data[0] > 60) {
            break;
        }
    }

    // closing library
    plcconnector_close();

    return 0;
}
