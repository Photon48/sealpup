# sealpup shell integration (bash)
# A binary can't change its parent shell's directory, so this function captures
# the path sealpup prints on `new`/`enter` and performs the cd itself. All other
# subcommands pass straight through, keeping `sealpup list | grep ...` working.
sealpup() {
  case "$1" in
    new|enter)
      local _sp_out
      _sp_out="$(_SEALPUP_SHIM=1 command sealpup "$@")" || return $?
      [ -n "$_sp_out" ] && builtin cd -- "$_sp_out"
      ;;
    *)
      command sealpup "$@"
      ;;
  esac
}
