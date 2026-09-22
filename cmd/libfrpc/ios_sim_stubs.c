// Copyright 2026 The frp Authors
//
// These symbols are referenced by the Go runtime when GOOS=darwin/arm64 is
// used to produce an iOS-simulator c-archive. The iOS SDK does not provide
// them; define them once (not in the cgo preamble, which is compiled into
// every cgo translation unit).

#include <TargetConditionals.h>

#if defined(TARGET_OS_SIMULATOR) && TARGET_OS_SIMULATOR
void darwin_arm_init_mach_exception_handler(void) {}
void darwin_arm_init_thread_exception_port(void) {}
#endif
