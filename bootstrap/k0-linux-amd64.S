/* Seme K0-P0 executor: Linux amd64, direct syscalls, no runtime. */
.global _start
.section .text

.equ SYS_READ,  0
.equ SYS_OPEN,  2
.equ SYS_CLOSE, 3
.equ SYS_LSEEK, 8
.equ SYS_EXIT, 60
.equ O_RDONLY, 0
.equ MAX_LOCALS, 256
.equ MAX_STACK, 1024
.equ MAX_INSTRUCTIONS, 65535
.equ MAX_FUNCTIONS, 16
.equ FRAME_LOCALS, 64
.equ MAX_CALL_DEPTH, 32

_start:
    mov (%rsp), %rax
    cmp $2, %rax
    jne invocation_error
    mov 16(%rsp), %rdi
    mov $SYS_OPEN, %rax
    mov $O_RDONLY, %rsi
    xor %rdx, %rdx
    syscall
    test %rax, %rax
    js io_error
    mov %rax, %r12

    lea magic(%rip), %r14
    mov $8, %r15d
magic_loop:
    call required_byte
    cmp (%r14), %al
    jne invalid
    inc %r14
    dec %r15d
    jnz magic_loop

    call read_uleb
    cmp $1, %rax
    je p0_header
    cmp $2, %rax
    je p1_header
    jmp invalid

p0_header:
    movq $1, image_profile(%rip)
    call read_uleb
    cmp $MAX_LOCALS, %rax
    ja invalid
    mov %rax, current_local_count(%rip)
    call read_uleb
    test %rax, %rax
    jz invalid
    cmp $MAX_INSTRUCTIONS, %rax
    ja invalid
    mov %rax, instruction_remaining(%rip)
    mov %rax, instruction_total(%rip)

    mov $SYS_LSEEK, %rax
    mov %r12, %rdi
    xor %rsi, %rsi
    mov $1, %rdx                    /* SEEK_CUR */
    syscall
    test %rax, %rax
    js close_io_error
    mov %rax, code_file_offset(%rip)

    movq $0, current_local_base(%rip)
    movq $0, current_stack_base(%rip)
    movq $0, call_depth(%rip)
    jmp initialize_frame

p1_header:
    movq $2, image_profile(%rip)
    call read_uleb
    test %rax, %rax
    jz invalid
    cmp $MAX_FUNCTIONS, %rax
    ja invalid
    mov %rax, function_count(%rip)
    xor %r13d, %r13d

p1_function_loop:
    cmp function_count(%rip), %r13
    jae p1_functions_done
    call read_uleb
    cmp $FRAME_LOCALS, %rax
    ja invalid
    lea function_params(%rip), %r11
    mov %rax, (%r11,%r13,8)
    mov %rax, %r14
    call read_uleb
    cmp $FRAME_LOCALS, %rax
    ja invalid
    cmp %rax, %r14
    ja invalid
    lea function_locals(%rip), %r11
    mov %rax, (%r11,%r13,8)
    call read_uleb
    test %rax, %rax
    jz invalid
    cmp $MAX_INSTRUCTIONS, %rax
    ja invalid
    lea function_instructions(%rip), %r11
    mov %rax, (%r11,%r13,8)
    call read_uleb
    test %rax, %rax
    jz invalid
    mov %rax, %r14
    mov $SYS_LSEEK, %rax
    mov %r12, %rdi
    xor %rsi, %rsi
    mov $1, %rdx
    syscall
    test %rax, %rax
    js close_io_error
    lea function_offsets(%rip), %r11
    mov %rax, (%r11,%r13,8)
    mov $SYS_LSEEK, %rax
    mov %r12, %rdi
    mov %r14, %rsi
    mov $1, %rdx
    syscall
    test %rax, %rax
    js close_io_error
    inc %r13
    jmp p1_function_loop

p1_functions_done:
    call read_uleb
    cmp function_count(%rip), %rax
    jae invalid
    mov %rax, current_function(%rip)
    mov %rax, %r13
    lea function_params(%rip), %r11
    cmpq $0, (%r11,%r13,8)
    jne invalid
    call read_byte
    test %edx, %edx
    jnz invalid
    movq $0, call_depth(%rip)
    movq $0, current_local_base(%rip)
    movq $0, current_stack_base(%rip)
    call enter_current_function

initialize_frame:

    lea local_initialized(%rip), %rdi
    mov $MAX_LOCALS, %rcx
    xor %eax, %eax
    rep stosb
    xor %ebp, %ebp                 /* stack depth */

execute:
    cmpq $0, instruction_remaining(%rip)
    je invalid
    decq instruction_remaining(%rip)
    call required_byte
    cmp $0x00, %al
    je op_return
    cmp $0x01, %al
    je op_const
    cmp $0x02, %al
    je op_local_get
    cmp $0x03, %al
    je op_local_set
    cmp $0x04, %al
    je op_drop
    cmp $0x05, %al
    je op_dup
    cmp $0x10, %al
    je op_add
    cmp $0x11, %al
    je op_sub
    cmp $0x12, %al
    je op_mul
    cmp $0x1a, %al
    je op_eq
    cmp $0x1b, %al
    je op_lt
    cmp $0x20, %al
    je op_jump
    cmp $0x21, %al
    je op_branch_false
    cmp $0x30, %al
    je op_call
    jmp invalid

op_const:
    call read_uleb
    mov %rax, %rbx
    call push_rbx
    jmp execute

op_local_get:
    call read_local_index
    add current_local_base(%rip), %rax
    lea local_initialized(%rip), %r11
    cmpb $0, (%r11,%rax)
    je invalid
    lea local_values(%rip), %r11
    mov (%r11,%rax,8), %rbx
    call push_rbx
    jmp execute

op_local_set:
    call read_local_index
    add current_local_base(%rip), %rax
    mov %rax, %r10
    call pop_rbx
    lea local_values(%rip), %r11
    mov %rbx, (%r11,%r10,8)
    lea local_initialized(%rip), %r11
    movb $1, (%r11,%r10)
    jmp execute

op_drop:
    call pop_rbx
    jmp execute

op_dup:
    call pop_rbx
    call push_rbx
    call push_rbx
    jmp execute

op_add:
    call pop_binary
    add %r10, %rbx
    call push_rbx
    jmp execute

op_sub:
    call pop_binary
    sub %r10, %rbx
    call push_rbx
    jmp execute

op_mul:
    call pop_binary
    imul %r10, %rbx
    call push_rbx
    jmp execute

op_eq:
    call pop_binary
    cmp %r10, %rbx
    sete %bl
    movzbl %bl, %ebx
    call push_rbx
    jmp execute

op_lt:
    call pop_binary
    cmp %r10, %rbx
    setb %bl
    movzbl %bl, %ebx
    call push_rbx
    jmp execute

op_jump:
    call read_branch_target
    call take_branch
    jmp execute

op_branch_false:
    call read_branch_target
    mov %rax, %r10
    mov %rbx, %r14
    call pop_rbx
    test %rbx, %rbx
    jnz execute
    mov %r10, %rax
    mov %r14, %rbx
    call take_branch
    jmp execute

op_call:
    cmpq $2, image_profile(%rip)
    jne invalid
    call read_uleb
    cmp function_count(%rip), %rax
    jae invalid
    mov %rax, %r14                  /* callee */
    mov call_depth(%rip), %r10
    cmp $(MAX_CALL_DEPTH - 1), %r10
    jae invalid

    /* Save the caller cursor and execution state. */
    mov $SYS_LSEEK, %rax
    mov %r12, %rdi
    xor %rsi, %rsi
    mov $1, %rdx
    syscall
    test %rax, %rax
    js close_io_error
    lea frame_return_offset(%rip), %r11
    mov %rax, (%r11,%r10,8)
    lea frame_remaining(%rip), %r11
    mov instruction_remaining(%rip), %rax
    mov %rax, (%r11,%r10,8)
    lea frame_function(%rip), %r11
    mov current_function(%rip), %rax
    mov %rax, (%r11,%r10,8)
    lea frame_stack_base(%rip), %r11
    mov current_stack_base(%rip), %rax
    mov %rax, (%r11,%r10,8)

    inc %r10
    mov %r10, call_depth(%rip)
    mov %r14, current_function(%rip)
    mov %r10, %rax
    imul $FRAME_LOCALS, %rax, %rax
    mov %rax, current_local_base(%rip)

    lea function_params(%rip), %r11
    mov (%r11,%r14,8), %r15
    mov %rbp, %rax
    sub %r15, %rax
    jc invalid
    mov %rax, current_stack_base(%rip)
    call clear_current_locals

    /* Pop arguments right-to-left into parameter locals. */
    test %r15, %r15
    jz call_enter
call_argument_loop:
    dec %r15
    call pop_rbx
    mov current_local_base(%rip), %rax
    add %r15, %rax
    lea local_values(%rip), %r11
    mov %rbx, (%r11,%rax,8)
    lea local_initialized(%rip), %r11
    movb $1, (%r11,%rax)
    test %r15, %r15
    jnz call_argument_loop
call_enter:
    call enter_current_function
    jmp execute

read_branch_target:
    call read_uleb
    cmp instruction_total(%rip), %rax
    jae invalid
    push %rax
    call read_uleb
    mov %rax, %rbx
    pop %rax
    ret

take_branch:
    mov instruction_total(%rip), %r10
    sub %rax, %r10
    mov %r10, instruction_remaining(%rip)
    add code_file_offset(%rip), %rbx
    jc invalid
    mov $SYS_LSEEK, %rax
    mov %r12, %rdi
    mov %rbx, %rsi
    xor %rdx, %rdx                  /* SEEK_SET */
    syscall
    cmp %rbx, %rax
    jne close_io_error
    ret

op_return:
    cmpq $0, instruction_remaining(%rip)
    jne invalid
    mov current_stack_base(%rip), %rax
    inc %rax
    cmp %rax, %rbp
    jne invalid
    call pop_rbx
    cmpq $0, call_depth(%rip)
    jne return_to_caller
    call push_rbx
    cmpq $1, image_profile(%rip)
    jne finish_entry
    call read_byte
    test %edx, %edx
    jnz invalid
finish_entry:
    call pop_rbx
    movzbl %bl, %edi
    jmp close_and_exit

return_to_caller:
    call push_rbx
    mov call_depth(%rip), %r10
    dec %r10
    mov %r10, call_depth(%rip)
    lea frame_remaining(%rip), %r11
    mov (%r11,%r10,8), %rax
    mov %rax, instruction_remaining(%rip)
    lea frame_function(%rip), %r11
    mov (%r11,%r10,8), %rax
    mov %rax, current_function(%rip)
    lea frame_stack_base(%rip), %r11
    mov (%r11,%r10,8), %rax
    mov %rax, current_stack_base(%rip)
    mov %r10, %rax
    imul $FRAME_LOCALS, %rax, %rax
    mov %rax, current_local_base(%rip)
    mov current_function(%rip), %rax
    lea function_locals(%rip), %r11
    mov (%r11,%rax,8), %rax
    mov %rax, current_local_count(%rip)
    lea frame_return_offset(%rip), %r11
    mov (%r11,%r10,8), %rsi
    mov $SYS_LSEEK, %rax
    mov %r12, %rdi
    xor %rdx, %rdx
    syscall
    cmp %rsi, %rax
    jne close_io_error
    jmp execute

enter_current_function:
    mov current_function(%rip), %rax
    lea function_locals(%rip), %r11
    mov (%r11,%rax,8), %r10
    mov %r10, current_local_count(%rip)
    lea function_instructions(%rip), %r11
    mov (%r11,%rax,8), %r10
    mov %r10, instruction_total(%rip)
    mov %r10, instruction_remaining(%rip)
    lea function_offsets(%rip), %r11
    mov (%r11,%rax,8), %rbx
    mov %rbx, code_file_offset(%rip)
    mov $SYS_LSEEK, %rax
    mov %r12, %rdi
    mov %rbx, %rsi
    xor %rdx, %rdx
    syscall
    cmp %rbx, %rax
    jne close_io_error
    ret

clear_current_locals:
    lea local_initialized(%rip), %rdi
    add current_local_base(%rip), %rdi
    mov $FRAME_LOCALS, %rcx
    xor %eax, %eax
    rep stosb
    ret

read_local_index:
    call read_uleb
    cmp current_local_count(%rip), %rax
    jae invalid
    ret

pop_binary:
    call pop_rbx                     /* right */
    mov %rbx, %r10
    call pop_rbx                     /* left */
    ret

push_rbx:
    cmp $MAX_STACK, %ebp
    jae invalid
    lea value_stack(%rip), %r11
    mov %rbx, (%r11,%rbp,8)
    inc %ebp
    ret

pop_rbx:
    test %ebp, %ebp
    jz invalid
    dec %ebp
    lea value_stack(%rip), %r11
    mov (%r11,%rbp,8), %rbx
    ret

/* Canonical uint64 ULEB128. */
read_uleb:
    xor %ebx, %ebx                   /* value */
    xor %r15d, %r15d                 /* shift; syscall preserves r15 */
    xor %r8d, %r8d                   /* byte count */
uleb_loop:
    call required_byte
    mov %eax, %r9d
    and $0x7f, %r9d
    cmp $63, %r15d
    jne uleb_shift
    cmp $1, %r9d
    ja invalid
uleb_shift:
    mov %r15d, %ecx
    shl %cl, %r9
    or %r9, %rbx
    inc %r8d
    test $0x80, %al
    jz uleb_done
    add $7, %r15d
    cmp $63, %r15d
    ja invalid
    jmp uleb_loop
uleb_done:
    cmp $1, %r8d
    je uleb_ok
    test %r9d, %r9d
    jz invalid                       /* non-minimal terminal zero */
uleb_ok:
    mov %rbx, %rax
    ret

required_byte:
    call read_byte
    test %edx, %edx
    jz invalid
    ret

read_byte:
    mov $SYS_READ, %rax
    mov %r12, %rdi
    lea input_byte(%rip), %rsi
    mov $1, %rdx
    syscall
    test %rax, %rax
    js close_io_error
    mov %eax, %edx
    jz read_done
    movzbl input_byte(%rip), %eax
read_done:
    ret

invalid:
    mov $65, %edi
    jmp close_and_exit
close_io_error:
    mov $74, %edi
close_and_exit:
    push %rdi
    mov $SYS_CLOSE, %rax
    mov %r12, %rdi
    syscall
    pop %rdi
    jmp exit
io_error:
    mov $74, %edi
    jmp exit
invocation_error:
    mov $64, %edi
exit:
    mov $SYS_EXIT, %rax
    syscall

.section .rodata
magic:
    .ascii "SEME-K0"
    .byte 0

.section .bss
input_byte:
    .skip 1
.balign 8
instruction_remaining:
    .skip 8
instruction_total:
    .skip 8
code_file_offset:
    .skip 8
image_profile:
    .skip 8
function_count:
    .skip 8
current_function:
    .skip 8
current_local_count:
    .skip 8
current_local_base:
    .skip 8
current_stack_base:
    .skip 8
call_depth:
    .skip 8
function_params:
    .skip MAX_FUNCTIONS * 8
function_locals:
    .skip MAX_FUNCTIONS * 8
function_instructions:
    .skip MAX_FUNCTIONS * 8
function_offsets:
    .skip MAX_FUNCTIONS * 8
frame_return_offset:
    .skip MAX_CALL_DEPTH * 8
frame_remaining:
    .skip MAX_CALL_DEPTH * 8
frame_function:
    .skip MAX_CALL_DEPTH * 8
frame_stack_base:
    .skip MAX_CALL_DEPTH * 8
local_values:
    .skip MAX_CALL_DEPTH * FRAME_LOCALS * 8
local_initialized:
    .skip MAX_CALL_DEPTH * FRAME_LOCALS
.balign 8
value_stack:
    .skip MAX_STACK * 8
