/* Seme Hex0 seed: Linux amd64, System V process entry, direct syscalls only. */
.global _start
.section .text

.equ SYS_READ,   0
.equ SYS_WRITE,  1
.equ SYS_OPEN,   2
.equ SYS_CLOSE,  3
.equ SYS_EXIT,  60
.equ O_RDONLY,   0
.equ O_WRONLY_CREAT_TRUNC, 577
.equ OUTPUT_MODE, 493              /* 0755 */

_start:
    mov (%rsp), %rax
    cmp $3, %rax
    jne invocation_error

    mov 16(%rsp), %rdi
    mov $SYS_OPEN, %rax
    mov $O_RDONLY, %rsi
    xor %rdx, %rdx
    syscall
    test %rax, %rax
    js io_error
    mov %rax, %r12                  /* input fd */

    mov 24(%rsp), %rdi
    mov $SYS_OPEN, %rax
    mov $O_WRONLY_CREAT_TRUNC, %rsi
    mov $OUTPUT_MODE, %rdx
    syscall
    test %rax, %rax
    js close_input_io_error
    mov %rax, %r13                  /* output fd */

    xor %r14d, %r14d                /* have high nibble */
    xor %r15d, %r15d                /* inside comment */

read_more:
    mov $SYS_READ, %rax
    mov %r12, %rdi
    lea input_buffer(%rip), %rsi
    mov $4096, %rdx
    syscall
    test %rax, %rax
    js close_both_io_error
    jz end_of_input
    lea input_buffer(%rip), %r9
    lea (%r9,%rax), %r8

parse_byte:
    cmp %r8, %r9
    je read_more
    movzbl (%r9), %eax
    inc %r9

    test %r15d, %r15d
    jz not_comment
    cmp $10, %al
    jne parse_byte
    xor %r15d, %r15d
    jmp parse_byte

not_comment:
    cmp $'#', %al
    je begin_comment
    cmp $' ', %al
    je parse_byte
    cmp $9, %al
    je parse_byte
    cmp $10, %al
    je parse_byte
    cmp $13, %al
    je parse_byte

    cmp $'0', %al
    jb invalid_hex
    cmp $'9', %al
    jbe decimal_digit
    cmp $'A', %al
    jb maybe_lower
    cmp $'F', %al
    jbe upper_digit
maybe_lower:
    cmp $'a', %al
    jb invalid_hex
    cmp $'f', %al
    ja invalid_hex
    sub $'a' - 10, %al
    jmp accept_nibble
upper_digit:
    sub $'A' - 10, %al
    jmp accept_nibble
decimal_digit:
    sub $'0', %al

accept_nibble:
    test %r14d, %r14d
    jnz emit_byte
    shl $4, %eax
    mov %eax, %ebx
    mov $1, %r14d
    jmp parse_byte

emit_byte:
    or %eax, %ebx
    mov %bl, output_byte(%rip)
    mov $SYS_WRITE, %rax
    mov %r13, %rdi
    lea output_byte(%rip), %rsi
    mov $1, %rdx
    syscall
    cmp $1, %rax
    jne close_both_io_error
    xor %r14d, %r14d
    jmp parse_byte

begin_comment:
    mov $1, %r15d
    jmp parse_byte

end_of_input:
    test %r14d, %r14d
    jnz invalid_hex
    mov $SYS_CLOSE, %rax
    mov %r12, %rdi
    syscall
    mov $SYS_CLOSE, %rax
    mov %r13, %rdi
    syscall
    xor %rdi, %rdi
    jmp exit

invalid_hex:
    mov $65, %rdi
    jmp close_both

close_both_io_error:
    mov $74, %rdi
close_both:
    push %rdi
    mov $SYS_CLOSE, %rax
    mov %r12, %rdi
    syscall
    mov $SYS_CLOSE, %rax
    mov %r13, %rdi
    syscall
    pop %rdi
    jmp exit

close_input_io_error:
    mov $SYS_CLOSE, %rax
    mov %r12, %rdi
    syscall
io_error:
    mov $74, %rdi
    jmp exit

invocation_error:
    mov $64, %rdi
exit:
    mov $SYS_EXIT, %rax
    syscall

.section .bss
.balign 16
input_buffer:
    .skip 4096
output_byte:
    .skip 1
