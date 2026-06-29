#include "../../include/rivex_ffi.h"
#include "_cgo_export.h"

RivexResponseData* go_handler_bridge(const RivexRequestData *request, void *user_data) {
    return goHandlerTrampoline((RivexRequestData*)request, user_data);
}
