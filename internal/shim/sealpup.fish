# sealpup shell integration (fish)
# A binary can't change its parent shell's directory, so this function captures
# the path sealpup prints on `new`/`enter` and performs the cd itself. All other
# subcommands pass straight through, keeping `sealpup list | grep ...` working.
function sealpup
    switch "$argv[1]"
        case new enter
            set -lx _SEALPUP_SHIM 1
            set -l _sp_out (command sealpup $argv)
            set -l _sp_code $status
            if test $_sp_code -ne 0
                return $_sp_code
            end
            if test -n "$_sp_out"
                builtin cd -- $_sp_out
            end
        case '*'
            command sealpup $argv
    end
end
