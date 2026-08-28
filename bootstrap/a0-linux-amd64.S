/* Seme A0 seed: bHH and uDECIMAL to bytes, Linux amd64 direct syscalls. */
.global _start
.section .text

.equ SYS_READ,   0
.equ SYS_WRITE,  1
.equ SYS_OPEN,   2
.equ SYS_CLOSE,  3
.equ SYS_EXIT,  60
.equ O_RDONLY,   0
.equ O_WRONLY_CREAT_TRUNC, 577
.equ OUTPUT_MODE, 493

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
    mov %rax, %r12

    mov 24(%rsp), %rdi
    mov $SYS_OPEN, %rax
    mov $O_WRONLY_CREAT_TRUNC, %rsi
    mov $OUTPUT_MODE, %rdx
    syscall
    test %rax, %rax
    js close_input_io_error
    mov %rax, %r13

directive:
    call next_token
    test %edx, %edx
    jz success
    cmp $'b', %al
    je byte_directive
    cmp $'u', %al
    je unsigned_directive
    jmp invalid

byte_directive:
    call required_char
    call hex_nibble
    shl $4, %eax
    mov %eax, %ebx
    call required_char
    call hex_nibble
    or %eax, %ebx
    call require_end
    mov %bl, output_byte(%rip)
    call write_byte
    jmp directive

unsigned_directive:
    xor %ebx, %ebx
    xor %r14d, %r14d
decimal_loop:
    call read_char
    test %edx, %edx
    jz decimal_eof
    cmp $'0', %al
    jb decimal_end
    cmp $'9', %al
    ja decimal_end
    sub $'0', %al
    movzbl %al, %ecx
    imul $10, %rbx, %rbx
    jo invalid
    add %rcx, %rbx
    jc invalid
    mov $1, %r14d
    jmp decimal_loop

decimal_end:
    test %r14d, %r14d
    jz invalid
    call accept_delimiter
    call emit_uleb
    jmp directive
decimal_eof:
    test %r14d, %r14d
    jz invalid
    call emit_uleb
    jmp success

emit_uleb:
    mov %rbx, %rax
    and $0x7f, %eax
    shr $7, %rbx
    test %rbx, %rbx
    jz emit_uleb_last
    or $0x80, %eax
emit_uleb_last:
    mov %al, output_byte(%rip)
    call write_byte
    test %rbx, %rbx
    jnz emit_uleb
    ret

next_token:
next_again:
    call read_char
    test %edx, %edx
    jz next_done
    cmp $'#', %al
    je skip_comment
    call is_whitespace
    test %ecx, %ecx
    jnz next_again
next_done:
    ret

skip_comment:
    call read_char
    test %edx, %edx
    jz next_done
    cmp $10, %al
    jne skip_comment
    jmp next_again

required_char:
    call read_char
    test %edx, %edx
    jz invalid
    ret

require_end:
    call read_char
    test %edx, %edx
    jz success_after_byte
    call accept_delimiter
    ret
success_after_byte:
    mov %bl, output_byte(%rip)
    call write_byte
    jmp success

accept_delimiter:
    cmp $'#', %al
    je consume_comment
    call is_whitespace
    test %ecx, %ecx
    jz invalid
    ret
consume_comment:
    call read_char
    test %edx, %edx
    jz delimiter_done
    cmp $10, %al
    jne consume_comment
delimiter_done:
    ret

is_whitespace:
    xor %ecx, %ecx
    cmp $' ', %al
    je whitespace_yes
    cmp $9, %al
    je whitespace_yes
    cmp $10, %al
    je whitespace_yes
    cmp $13, %al
    jne whitespace_done
whitespace_yes:
    mov $1, %ecx
whitespace_done:
    ret

hex_nibble:
    cmp $'0', %al
    jb invalid
    cmp $'9', %al
    jbe hex_decimal
    cmp $'A', %al
    jb hex_lower
    cmp $'F', %al
    jbe hex_upper
hex_lower:
    cmp $'a', %al
    jb invalid
    cmp $'f', %al
    ja invalid
    sub $'a' - 10, %al
    ret
hex_upper:
    sub $'A' - 10, %al
    ret
hex_decimal:
    sub $'0', %al
    ret

read_char:
    mov $SYS_READ, %rax
    mov %r12, %rdi
    lea input_byte(%rip), %rsi
    mov $1, %rdx
    syscall
    test %rax, %rax
    js close_both_io_error
    mov %eax, %edx
    jz read_done
    movzbl input_byte(%rip), %eax
read_done:
    ret

write_byte:
    mov $SYS_WRITE, %rax
    mov %r13, %rdi
    lea output_byte(%rip), %rsi
    mov $1, %rdx
    syscall
    cmp $1, %rax
    jne close_both_io_error
    ret

success:
    xor %edi, %edi
    jmp close_both
invalid:
    mov $65, %edi
    jmp close_both
close_both_io_error:
    mov $74, %edi
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
    mov $74, %edi
    jmp exit
invocation_error:
    mov $64, %edi
exit:
    mov $SYS_EXIT, %rax
    syscall

.section .bss
input_byte:
    .skip 1
output_byte:
    .skip 1

