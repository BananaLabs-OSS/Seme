/* Seme K0-P0 executor: Linux amd64, direct syscalls, no runtime. */
.global _start
.section .text

.equ SYS_READ,  0
.equ SYS_OPEN,  2
.equ SYS_CLOSE, 3
.equ SYS_EXIT, 60
.equ O_RDONLY, 0
.equ MAX_LOCALS, 256
.equ MAX_STACK, 1024
.equ MAX_INSTRUCTIONS, 65535

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
    jne invalid
    call read_uleb
    cmp $MAX_LOCALS, %rax
    ja invalid
    mov %rax, %r13
    call read_uleb
    test %rax, %rax
    jz invalid
    cmp $MAX_INSTRUCTIONS, %rax
    ja invalid
    mov %rax, instruction_remaining(%rip)

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
    jmp invalid

op_const:
    call read_uleb
    mov %rax, %rbx
    call push_rbx
    jmp execute

op_local_get:
    call read_local_index
    lea local_initialized(%rip), %r11
    cmpb $0, (%r11,%rax)
    je invalid
    lea local_values(%rip), %r11
    mov (%r11,%rax,8), %rbx
    call push_rbx
    jmp execute

op_local_set:
    call read_local_index
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

op_return:
    cmpq $0, instruction_remaining(%rip)
    jne invalid
    cmp $1, %ebp
    jne invalid
    call pop_rbx
    call read_byte
    test %edx, %edx
    jnz invalid
    movzbl %bl, %edi
    jmp close_and_exit

read_local_index:
    call read_uleb
    cmp %r13, %rax
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
local_values:
    .skip MAX_LOCALS * 8
local_initialized:
    .skip MAX_LOCALS
.balign 8
value_stack:
    .skip MAX_STACK * 8
