# bash completion for devspace                             -*- shell-script -*-

__devspace_debug()
{
    if [[ -n ${BASH_COMP_DEBUG_FILE} ]]; then
        echo "$*" >> "${BASH_COMP_DEBUG_FILE}"
    fi
}

# Homebrew on Macs have version 1.3 of bash-completion which doesn't include
# _init_completion. This is a very minimal version of that function.
__devspace_init_completion()
{
    COMPREPLY=()
    _get_comp_words_by_ref "$@" cur prev words cword
}

__devspace_index_of_word()
{
    local w word=$1
    shift
    index=0
    for w in "$@"; do
        [[ $w = "$word" ]] && return
        index=$((index+1))
    done
    index=-1
}

__devspace_contains_word()
{
    local w word=$1; shift
    for w in "$@"; do
        [[ $w = "$word" ]] && return
    done
    return 1
}

__devspace_handle_go_custom_completion()
{
    __devspace_debug "${FUNCNAME[0]}: cur is ${cur}, words[*] is ${words[*]}, #words[@] is ${#words[@]}"

    local out requestComp lastParam lastChar comp directive args

    # Prepare the command to request completions for the program.
    # Calling ${words[0]} instead of directly devspace allows to handle aliases
    args=("${words[@]:1}")
    requestComp="${words[0]} __completeNoDesc ${args[*]}"

    lastParam=${words[$((${#words[@]}-1))]}
    lastChar=${lastParam:$((${#lastParam}-1)):1}
    __devspace_debug "${FUNCNAME[0]}: lastParam ${lastParam}, lastChar ${lastChar}"

    if [ -z "${cur}" ] && [ "${lastChar}" != "=" ]; then
        # If the last parameter is complete (there is a space following it)
        # We add an extra empty parameter so we can indicate this to the go method.
        __devspace_debug "${FUNCNAME[0]}: Adding extra empty parameter"
        requestComp="${requestComp} \"\""
    fi

    __devspace_debug "${FUNCNAME[0]}: calling ${requestComp}"
    # Use eval to handle any environment variables and such
    out=$(eval "${requestComp}" 2>/dev/null)

    # Extract the directive integer at the very end of the output following a colon (:)
    directive=${out##*:}
    # Remove the directive
    out=${out%:*}
    if [ "${directive}" = "${out}" ]; then
        # There is not directive specified
        directive=0
    fi
    __devspace_debug "${FUNCNAME[0]}: the completion directive is: ${directive}"
    __devspace_debug "${FUNCNAME[0]}: the completions are: ${out[*]}"

    if [ $((directive & 1)) -ne 0 ]; then
        # Error code.  No completion.
        __devspace_debug "${FUNCNAME[0]}: received error from custom completion go code"
        return
    else
        if [ $((directive & 2)) -ne 0 ]; then
            if [[ $(type -t compopt) = "builtin" ]]; then
                __devspace_debug "${FUNCNAME[0]}: activating no space"
                compopt -o nospace
            fi
        fi
        if [ $((directive & 4)) -ne 0 ]; then
            if [[ $(type -t compopt) = "builtin" ]]; then
                __devspace_debug "${FUNCNAME[0]}: activating no file completion"
                compopt +o default
            fi
        fi

        while IFS='' read -r comp; do
            COMPREPLY+=("$comp")
        done < <(compgen -W "${out[*]}" -- "$cur")
    fi
}

__devspace_handle_reply()
{
    __devspace_debug "${FUNCNAME[0]}"
    local comp
    case $cur in
        -*)
            if [[ $(type -t compopt) = "builtin" ]]; then
                compopt -o nospace
            fi
            local allflags
            if [ ${#must_have_one_flag[@]} -ne 0 ]; then
                allflags=("${must_have_one_flag[@]}")
            else
                allflags=("${flags[*]} ${two_word_flags[*]}")
            fi
            while IFS='' read -r comp; do
                COMPREPLY+=("$comp")
            done < <(compgen -W "${allflags[*]}" -- "$cur")
            if [[ $(type -t compopt) = "builtin" ]]; then
                [[ "${COMPREPLY[0]}" == *= ]] || compopt +o nospace
            fi

            # complete after --flag=abc
            if [[ $cur == *=* ]]; then
                if [[ $(type -t compopt) = "builtin" ]]; then
                    compopt +o nospace
                fi

                local index flag
                flag="${cur%=*}"
                __devspace_index_of_word "${flag}" "${flags_with_completion[@]}"
                COMPREPLY=()
                if [[ ${index} -ge 0 ]]; then
                    PREFIX=""
                    cur="${cur#*=}"
                    ${flags_completion[${index}]}
                    if [ -n "${ZSH_VERSION}" ]; then
                        # zsh completion needs --flag= prefix
                        eval "COMPREPLY=( \"\${COMPREPLY[@]/#/${flag}=}\" )"
                    fi
                fi
            fi
            return 0;
            ;;
    esac

    # check if we are handling a flag with special work handling
    local index
    __devspace_index_of_word "${prev}" "${flags_with_completion[@]}"
    if [[ ${index} -ge 0 ]]; then
        ${flags_completion[${index}]}
        return
    fi

    # we are parsing a flag and don't have a special handler, no completion
    if [[ ${cur} != "${words[cword]}" ]]; then
        return
    fi

    local completions
    completions=("${commands[@]}")
    if [[ ${#must_have_one_noun[@]} -ne 0 ]]; then
        completions=("${must_have_one_noun[@]}")
    elif [[ -n "${has_completion_function}" ]]; then
        # if a go completion function is provided, defer to that function
        completions=()
        __devspace_handle_go_custom_completion
    fi
    if [[ ${#must_have_one_flag[@]} -ne 0 ]]; then
        completions+=("${must_have_one_flag[@]}")
    fi
    while IFS='' read -r comp; do
        COMPREPLY+=("$comp")
    done < <(compgen -W "${completions[*]}" -- "$cur")

    if [[ ${#COMPREPLY[@]} -eq 0 && ${#noun_aliases[@]} -gt 0 && ${#must_have_one_noun[@]} -ne 0 ]]; then
        while IFS='' read -r comp; do
            COMPREPLY+=("$comp")
        done < <(compgen -W "${noun_aliases[*]}" -- "$cur")
    fi

    if [[ ${#COMPREPLY[@]} -eq 0 ]]; then
		if declare -F __devspace_custom_func >/dev/null; then
			# try command name qualified custom func
			__devspace_custom_func
		else
			# otherwise fall back to unqualified for compatibility
			declare -F __custom_func >/dev/null && __custom_func
		fi
    fi

    # available in bash-completion >= 2, not always present on macOS
    if declare -F __ltrim_colon_completions >/dev/null; then
        __ltrim_colon_completions "$cur"
    fi

    # If there is only 1 completion and it is a flag with an = it will be completed
    # but we don't want a space after the =
    if [[ "${#COMPREPLY[@]}" -eq "1" ]] && [[ $(type -t compopt) = "builtin" ]] && [[ "${COMPREPLY[0]}" == --*= ]]; then
       compopt -o nospace
    fi
}

# The arguments should be in the form "ext1|ext2|extn"
__devspace_handle_filename_extension_flag()
{
    local ext="$1"
    _filedir "@(${ext})"
}

__devspace_handle_subdirs_in_dir_flag()
{
    local dir="$1"
    pushd "${dir}" >/dev/null 2>&1 && _filedir -d && popd >/dev/null 2>&1 || return
}

__devspace_handle_flag()
{
    __devspace_debug "${FUNCNAME[0]}: c is $c words[c] is ${words[c]}"

    # if a command required a flag, and we found it, unset must_have_one_flag()
    local flagname=${words[c]}
    local flagvalue
    # if the word contained an =
    if [[ ${words[c]} == *"="* ]]; then
        flagvalue=${flagname#*=} # take in as flagvalue after the =
        flagname=${flagname%=*} # strip everything after the =
        flagname="${flagname}=" # but put the = back
    fi
    __devspace_debug "${FUNCNAME[0]}: looking for ${flagname}"
    if __devspace_contains_word "${flagname}" "${must_have_one_flag[@]}"; then
        must_have_one_flag=()
    fi

    # if you set a flag which only applies to this command, don't show subcommands
    if __devspace_contains_word "${flagname}" "${local_nonpersistent_flags[@]}"; then
      commands=()
    fi

    # keep flag value with flagname as flaghash
    # flaghash variable is an associative array which is only supported in bash > 3.
    if [[ -z "${BASH_VERSION}" || "${BASH_VERSINFO[0]}" -gt 3 ]]; then
        if [ -n "${flagvalue}" ] ; then
            flaghash[${flagname}]=${flagvalue}
        elif [ -n "${words[ $((c+1)) ]}" ] ; then
            flaghash[${flagname}]=${words[ $((c+1)) ]}
        else
            flaghash[${flagname}]="true" # pad "true" for bool flag
        fi
    fi

    # skip the argument to a two word flag
    if [[ ${words[c]} != *"="* ]] && __devspace_contains_word "${words[c]}" "${two_word_flags[@]}"; then
			  __devspace_debug "${FUNCNAME[0]}: found a flag ${words[c]}, skip the next argument"
        c=$((c+1))
        # if we are looking for a flags value, don't show commands
        if [[ $c -eq $cword ]]; then
            commands=()
        fi
    fi

    c=$((c+1))

}

__devspace_handle_noun()
{
    __devspace_debug "${FUNCNAME[0]}: c is $c words[c] is ${words[c]}"

    if __devspace_contains_word "${words[c]}" "${must_have_one_noun[@]}"; then
        must_have_one_noun=()
    elif __devspace_contains_word "${words[c]}" "${noun_aliases[@]}"; then
        must_have_one_noun=()
    fi

    nouns+=("${words[c]}")
    c=$((c+1))
}

__devspace_handle_command()
{
    __devspace_debug "${FUNCNAME[0]}: c is $c words[c] is ${words[c]}"

    local next_command
    if [[ -n ${last_command} ]]; then
        next_command="_${last_command}_${words[c]//:/__}"
    else
        if [[ $c -eq 0 ]]; then
            next_command="_devspace_root_command"
        else
            next_command="_${words[c]//:/__}"
        fi
    fi
    c=$((c+1))
    __devspace_debug "${FUNCNAME[0]}: looking for ${next_command}"
    declare -F "$next_command" >/dev/null && $next_command
}

__devspace_handle_word()
{
    if [[ $c -ge $cword ]]; then
        __devspace_handle_reply
        return
    fi
    __devspace_debug "${FUNCNAME[0]}: c is $c words[c] is ${words[c]}"
    if [[ "${words[c]}" == -* ]]; then
        __devspace_handle_flag
    elif __devspace_contains_word "${words[c]}" "${commands[@]}"; then
        __devspace_handle_command
    elif [[ $c -eq 0 ]]; then
        __devspace_handle_command
    elif __devspace_contains_word "${words[c]}" "${command_aliases[@]}"; then
        # aliashash variable is an associative array which is only supported in bash > 3.
        if [[ -z "${BASH_VERSION}" || "${BASH_VERSINFO[0]}" -gt 3 ]]; then
            words[c]=${aliashash[${words[c]}]}
            __devspace_handle_command
        else
            __devspace_handle_noun
        fi
    else
        __devspace_handle_noun
    fi
    __devspace_handle_word
}

_devspace_add_deployment()
{
    last_command="devspace_add_deployment"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--chart=")
    two_word_flags+=("--chart")
    local_nonpersistent_flags+=("--chart=")
    flags+=("--chart-repo=")
    two_word_flags+=("--chart-repo")
    local_nonpersistent_flags+=("--chart-repo=")
    flags+=("--chart-version=")
    two_word_flags+=("--chart-version")
    local_nonpersistent_flags+=("--chart-version=")
    flags+=("--component=")
    two_word_flags+=("--component")
    local_nonpersistent_flags+=("--component=")
    flags+=("--context=")
    two_word_flags+=("--context")
    local_nonpersistent_flags+=("--context=")
    flags+=("--dockerfile=")
    two_word_flags+=("--dockerfile")
    local_nonpersistent_flags+=("--dockerfile=")
    flags+=("--image=")
    two_word_flags+=("--image")
    local_nonpersistent_flags+=("--image=")
    flags+=("--manifests=")
    two_word_flags+=("--manifests")
    local_nonpersistent_flags+=("--manifests=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_add_image()
{
    last_command="devspace_add_image"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--buildtool=")
    two_word_flags+=("--buildtool")
    local_nonpersistent_flags+=("--buildtool=")
    flags+=("--context=")
    two_word_flags+=("--context")
    local_nonpersistent_flags+=("--context=")
    flags+=("--dockerfile=")
    two_word_flags+=("--dockerfile")
    local_nonpersistent_flags+=("--dockerfile=")
    flags+=("--image=")
    two_word_flags+=("--image")
    local_nonpersistent_flags+=("--image=")
    flags+=("--tag=")
    two_word_flags+=("--tag")
    local_nonpersistent_flags+=("--tag=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_flag+=("--image=")
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_add_port()
{
    last_command="devspace_add_port"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--label-selector=")
    two_word_flags+=("--label-selector")
    local_nonpersistent_flags+=("--label-selector=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_add_provider()
{
    last_command="devspace_add_provider"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--host=")
    two_word_flags+=("--host")
    local_nonpersistent_flags+=("--host=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_add_sync()
{
    last_command="devspace_add_sync"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--container=")
    two_word_flags+=("--container")
    local_nonpersistent_flags+=("--container=")
    flags+=("--exclude=")
    two_word_flags+=("--exclude")
    local_nonpersistent_flags+=("--exclude=")
    flags+=("--label-selector=")
    two_word_flags+=("--label-selector")
    local_nonpersistent_flags+=("--label-selector=")
    flags+=("--local=")
    two_word_flags+=("--local")
    local_nonpersistent_flags+=("--local=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_flag+=("--container=")
    must_have_one_flag+=("--local=")
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_add()
{
    last_command="devspace_add"

    command_aliases=()

    commands=()
    commands+=("deployment")
    commands+=("image")
    commands+=("port")
    commands+=("provider")
    commands+=("sync")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_analyze()
{
    last_command="devspace_analyze"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--patient")
    local_nonpersistent_flags+=("--patient")
    flags+=("--timeout=")
    two_word_flags+=("--timeout")
    local_nonpersistent_flags+=("--timeout=")
    flags+=("--wait")
    local_nonpersistent_flags+=("--wait")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_attach()
{
    last_command="devspace_attach"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--container=")
    two_word_flags+=("--container")
    two_word_flags+=("-c")
    local_nonpersistent_flags+=("--container=")
    flags+=("--image=")
    two_word_flags+=("--image")
    local_nonpersistent_flags+=("--image=")
    flags+=("--label-selector=")
    two_word_flags+=("--label-selector")
    two_word_flags+=("-l")
    local_nonpersistent_flags+=("--label-selector=")
    flags+=("--pick")
    local_nonpersistent_flags+=("--pick")
    flags+=("--pod=")
    two_word_flags+=("--pod")
    local_nonpersistent_flags+=("--pod=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_build()
{
    last_command="devspace_build"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--allow-cyclic")
    local_nonpersistent_flags+=("--allow-cyclic")
    flags+=("--build-sequential")
    local_nonpersistent_flags+=("--build-sequential")
    flags+=("--dependency=")
    two_word_flags+=("--dependency")
    local_nonpersistent_flags+=("--dependency=")
    flags+=("--force-build")
    flags+=("-b")
    local_nonpersistent_flags+=("--force-build")
    flags+=("--force-dependencies")
    local_nonpersistent_flags+=("--force-dependencies")
    flags+=("--skip-push")
    local_nonpersistent_flags+=("--skip-push")
    flags+=("--tag=")
    two_word_flags+=("--tag")
    two_word_flags+=("-t")
    local_nonpersistent_flags+=("--tag=")
    flags+=("--verbose-dependencies")
    local_nonpersistent_flags+=("--verbose-dependencies")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_cleanup_images()
{
    last_command="devspace_cleanup_images"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_cleanup()
{
    last_command="devspace_cleanup"

    command_aliases=()

    commands=()
    commands+=("images")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_connect_cluster()
{
    last_command="devspace_connect_cluster"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--admission-controller")
    local_nonpersistent_flags+=("--admission-controller")
    flags+=("--cert-manager")
    local_nonpersistent_flags+=("--cert-manager")
    flags+=("--context=")
    two_word_flags+=("--context")
    local_nonpersistent_flags+=("--context=")
    flags+=("--domain=")
    two_word_flags+=("--domain")
    local_nonpersistent_flags+=("--domain=")
    flags+=("--gatekeeper")
    local_nonpersistent_flags+=("--gatekeeper")
    flags+=("--gatekeeper-rules")
    local_nonpersistent_flags+=("--gatekeeper-rules")
    flags+=("--ingress-controller")
    local_nonpersistent_flags+=("--ingress-controller")
    flags+=("--key=")
    two_word_flags+=("--key")
    local_nonpersistent_flags+=("--key=")
    flags+=("--name=")
    two_word_flags+=("--name")
    local_nonpersistent_flags+=("--name=")
    flags+=("--open-ui")
    local_nonpersistent_flags+=("--open-ui")
    flags+=("--provider=")
    two_word_flags+=("--provider")
    local_nonpersistent_flags+=("--provider=")
    flags+=("--public")
    local_nonpersistent_flags+=("--public")
    flags+=("--use-domain")
    local_nonpersistent_flags+=("--use-domain")
    flags+=("--use-hostnetwork")
    local_nonpersistent_flags+=("--use-hostnetwork")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_connect()
{
    last_command="devspace_connect"

    command_aliases=()

    commands=()
    commands+=("cluster")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_create_space()
{
    last_command="devspace_create_space"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--active")
    local_nonpersistent_flags+=("--active")
    flags+=("--cluster=")
    two_word_flags+=("--cluster")
    local_nonpersistent_flags+=("--cluster=")
    flags+=("--provider=")
    two_word_flags+=("--provider")
    local_nonpersistent_flags+=("--provider=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_create()
{
    last_command="devspace_create"

    command_aliases=()

    commands=()
    commands+=("space")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_deploy()
{
    last_command="devspace_deploy"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--allow-cyclic")
    local_nonpersistent_flags+=("--allow-cyclic")
    flags+=("--build-sequential")
    local_nonpersistent_flags+=("--build-sequential")
    flags+=("--dependency=")
    two_word_flags+=("--dependency")
    local_nonpersistent_flags+=("--dependency=")
    flags+=("--deployments=")
    two_word_flags+=("--deployments")
    local_nonpersistent_flags+=("--deployments=")
    flags+=("--force-build")
    flags+=("-b")
    local_nonpersistent_flags+=("--force-build")
    flags+=("--force-dependencies")
    local_nonpersistent_flags+=("--force-dependencies")
    flags+=("--force-deploy")
    flags+=("-d")
    local_nonpersistent_flags+=("--force-deploy")
    flags+=("--skip-build")
    local_nonpersistent_flags+=("--skip-build")
    flags+=("--skip-push")
    local_nonpersistent_flags+=("--skip-push")
    flags+=("--timeout=")
    two_word_flags+=("--timeout")
    local_nonpersistent_flags+=("--timeout=")
    flags+=("--verbose-dependencies")
    local_nonpersistent_flags+=("--verbose-dependencies")
    flags+=("--wait")
    local_nonpersistent_flags+=("--wait")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_dev()
{
    last_command="devspace_dev"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--allow-cyclic")
    local_nonpersistent_flags+=("--allow-cyclic")
    flags+=("--build-sequential")
    local_nonpersistent_flags+=("--build-sequential")
    flags+=("--deployments=")
    two_word_flags+=("--deployments")
    local_nonpersistent_flags+=("--deployments=")
    flags+=("--exit-after-deploy")
    local_nonpersistent_flags+=("--exit-after-deploy")
    flags+=("--force-build")
    flags+=("-b")
    local_nonpersistent_flags+=("--force-build")
    flags+=("--force-dependencies")
    local_nonpersistent_flags+=("--force-dependencies")
    flags+=("--force-deploy")
    flags+=("-d")
    local_nonpersistent_flags+=("--force-deploy")
    flags+=("--interactive")
    flags+=("-i")
    local_nonpersistent_flags+=("--interactive")
    flags+=("--open")
    local_nonpersistent_flags+=("--open")
    flags+=("--portforwarding")
    local_nonpersistent_flags+=("--portforwarding")
    flags+=("--skip-build")
    local_nonpersistent_flags+=("--skip-build")
    flags+=("--skip-pipeline")
    flags+=("-x")
    local_nonpersistent_flags+=("--skip-pipeline")
    flags+=("--skip-push")
    local_nonpersistent_flags+=("--skip-push")
    flags+=("--sync")
    local_nonpersistent_flags+=("--sync")
    flags+=("--terminal")
    flags+=("-t")
    local_nonpersistent_flags+=("--terminal")
    flags+=("--timeout=")
    two_word_flags+=("--timeout")
    local_nonpersistent_flags+=("--timeout=")
    flags+=("--ui")
    local_nonpersistent_flags+=("--ui")
    flags+=("--verbose-dependencies")
    local_nonpersistent_flags+=("--verbose-dependencies")
    flags+=("--verbose-sync")
    local_nonpersistent_flags+=("--verbose-sync")
    flags+=("--wait")
    local_nonpersistent_flags+=("--wait")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_enter()
{
    last_command="devspace_enter"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--container=")
    two_word_flags+=("--container")
    two_word_flags+=("-c")
    local_nonpersistent_flags+=("--container=")
    flags+=("--image=")
    two_word_flags+=("--image")
    local_nonpersistent_flags+=("--image=")
    flags+=("--label-selector=")
    two_word_flags+=("--label-selector")
    two_word_flags+=("-l")
    local_nonpersistent_flags+=("--label-selector=")
    flags+=("--pick")
    local_nonpersistent_flags+=("--pick")
    flags+=("--pod=")
    two_word_flags+=("--pod")
    local_nonpersistent_flags+=("--pod=")
    flags+=("--wait")
    local_nonpersistent_flags+=("--wait")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_init()
{
    last_command="devspace_init"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--context=")
    two_word_flags+=("--context")
    local_nonpersistent_flags+=("--context=")
    flags+=("--dockerfile=")
    two_word_flags+=("--dockerfile")
    local_nonpersistent_flags+=("--dockerfile=")
    flags+=("--provider=")
    two_word_flags+=("--provider")
    local_nonpersistent_flags+=("--provider=")
    flags+=("--reconfigure")
    flags+=("-r")
    local_nonpersistent_flags+=("--reconfigure")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_list_clusters()
{
    last_command="devspace_list_clusters"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--all")
    local_nonpersistent_flags+=("--all")
    flags+=("--provider=")
    two_word_flags+=("--provider")
    local_nonpersistent_flags+=("--provider=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_list_commands()
{
    last_command="devspace_list_commands"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_list_contexts()
{
    last_command="devspace_list_contexts"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_list_deployments()
{
    last_command="devspace_list_deployments"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_list_namespaces()
{
    last_command="devspace_list_namespaces"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_list_ports()
{
    last_command="devspace_list_ports"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_list_profiles()
{
    last_command="devspace_list_profiles"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_list_providers()
{
    last_command="devspace_list_providers"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_list_spaces()
{
    last_command="devspace_list_spaces"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--all")
    local_nonpersistent_flags+=("--all")
    flags+=("--cluster=")
    two_word_flags+=("--cluster")
    local_nonpersistent_flags+=("--cluster=")
    flags+=("--name=")
    two_word_flags+=("--name")
    local_nonpersistent_flags+=("--name=")
    flags+=("--provider=")
    two_word_flags+=("--provider")
    local_nonpersistent_flags+=("--provider=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_list_sync()
{
    last_command="devspace_list_sync"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_list_vars()
{
    last_command="devspace_list_vars"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_list()
{
    last_command="devspace_list"

    command_aliases=()

    commands=()
    commands+=("clusters")
    commands+=("commands")
    commands+=("contexts")
    commands+=("deployments")
    commands+=("namespaces")
    commands+=("ports")
    commands+=("profiles")
    commands+=("providers")
    commands+=("spaces")
    commands+=("sync")
    commands+=("vars")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_login()
{
    last_command="devspace_login"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--key=")
    two_word_flags+=("--key")
    local_nonpersistent_flags+=("--key=")
    flags+=("--provider=")
    two_word_flags+=("--provider")
    local_nonpersistent_flags+=("--provider=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_logs()
{
    last_command="devspace_logs"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--container=")
    two_word_flags+=("--container")
    two_word_flags+=("-c")
    local_nonpersistent_flags+=("--container=")
    flags+=("--follow")
    flags+=("-f")
    local_nonpersistent_flags+=("--follow")
    flags+=("--image=")
    two_word_flags+=("--image")
    local_nonpersistent_flags+=("--image=")
    flags+=("--label-selector=")
    two_word_flags+=("--label-selector")
    two_word_flags+=("-l")
    local_nonpersistent_flags+=("--label-selector=")
    flags+=("--lines=")
    two_word_flags+=("--lines")
    local_nonpersistent_flags+=("--lines=")
    flags+=("--pick")
    local_nonpersistent_flags+=("--pick")
    flags+=("--pod=")
    two_word_flags+=("--pod")
    local_nonpersistent_flags+=("--pod=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_open()
{
    last_command="devspace_open"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--provider=")
    two_word_flags+=("--provider")
    local_nonpersistent_flags+=("--provider=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_print()
{
    last_command="devspace_print"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--skip-info")
    local_nonpersistent_flags+=("--skip-info")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_purge()
{
    last_command="devspace_purge"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--allow-cyclic")
    local_nonpersistent_flags+=("--allow-cyclic")
    flags+=("--dependencies")
    local_nonpersistent_flags+=("--dependencies")
    flags+=("--dependency=")
    two_word_flags+=("--dependency")
    local_nonpersistent_flags+=("--dependency=")
    flags+=("--deployments=")
    two_word_flags+=("--deployments")
    two_word_flags+=("-d")
    local_nonpersistent_flags+=("--deployments=")
    flags+=("--verbose-dependencies")
    local_nonpersistent_flags+=("--verbose-dependencies")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_remove_cluster()
{
    last_command="devspace_remove_cluster"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--provider=")
    two_word_flags+=("--provider")
    local_nonpersistent_flags+=("--provider=")
    flags+=("--yes")
    flags+=("-y")
    local_nonpersistent_flags+=("--yes")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_remove_context()
{
    last_command="devspace_remove_context"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--all-spaces")
    local_nonpersistent_flags+=("--all-spaces")
    flags+=("--provider=")
    two_word_flags+=("--provider")
    local_nonpersistent_flags+=("--provider=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_remove_deployment()
{
    last_command="devspace_remove_deployment"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--all")
    local_nonpersistent_flags+=("--all")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_remove_image()
{
    last_command="devspace_remove_image"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--all")
    local_nonpersistent_flags+=("--all")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_remove_port()
{
    last_command="devspace_remove_port"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--all")
    local_nonpersistent_flags+=("--all")
    flags+=("--label-selector=")
    two_word_flags+=("--label-selector")
    local_nonpersistent_flags+=("--label-selector=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_remove_provider()
{
    last_command="devspace_remove_provider"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--name=")
    two_word_flags+=("--name")
    local_nonpersistent_flags+=("--name=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_remove_space()
{
    last_command="devspace_remove_space"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--all")
    local_nonpersistent_flags+=("--all")
    flags+=("--id=")
    two_word_flags+=("--id")
    local_nonpersistent_flags+=("--id=")
    flags+=("--provider=")
    two_word_flags+=("--provider")
    local_nonpersistent_flags+=("--provider=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_remove_sync()
{
    last_command="devspace_remove_sync"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--all")
    local_nonpersistent_flags+=("--all")
    flags+=("--container=")
    two_word_flags+=("--container")
    local_nonpersistent_flags+=("--container=")
    flags+=("--label-selector=")
    two_word_flags+=("--label-selector")
    local_nonpersistent_flags+=("--label-selector=")
    flags+=("--local=")
    two_word_flags+=("--local")
    local_nonpersistent_flags+=("--local=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_remove()
{
    last_command="devspace_remove"

    command_aliases=()

    commands=()
    commands+=("cluster")
    commands+=("context")
    commands+=("deployment")
    commands+=("image")
    commands+=("port")
    commands+=("provider")
    commands+=("space")
    commands+=("sync")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_render()
{
    last_command="devspace_render"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--allow-cyclic")
    local_nonpersistent_flags+=("--allow-cyclic")
    flags+=("--build-sequential")
    local_nonpersistent_flags+=("--build-sequential")
    flags+=("--dependency=")
    two_word_flags+=("--dependency")
    local_nonpersistent_flags+=("--dependency=")
    flags+=("--deployments=")
    two_word_flags+=("--deployments")
    local_nonpersistent_flags+=("--deployments=")
    flags+=("--force-build")
    flags+=("-b")
    local_nonpersistent_flags+=("--force-build")
    flags+=("--show-logs")
    local_nonpersistent_flags+=("--show-logs")
    flags+=("--skip-build")
    local_nonpersistent_flags+=("--skip-build")
    flags+=("--skip-dependencies")
    local_nonpersistent_flags+=("--skip-dependencies")
    flags+=("--skip-push")
    local_nonpersistent_flags+=("--skip-push")
    flags+=("--tag=")
    two_word_flags+=("--tag")
    two_word_flags+=("-t")
    local_nonpersistent_flags+=("--tag=")
    flags+=("--verbose-dependencies")
    local_nonpersistent_flags+=("--verbose-dependencies")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_reset_dependencies()
{
    last_command="devspace_reset_dependencies"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_reset_key()
{
    last_command="devspace_reset_key"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--provider=")
    two_word_flags+=("--provider")
    local_nonpersistent_flags+=("--provider=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_reset_vars()
{
    last_command="devspace_reset_vars"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_reset()
{
    last_command="devspace_reset"

    command_aliases=()

    commands=()
    commands+=("dependencies")
    commands+=("key")
    commands+=("vars")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_run()
{
    last_command="devspace_run"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_set_analytics()
{
    last_command="devspace_set_analytics"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_set_encryptionkey()
{
    last_command="devspace_set_encryptionkey"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--cluster=")
    two_word_flags+=("--cluster")
    local_nonpersistent_flags+=("--cluster=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_set_var()
{
    last_command="devspace_set_var"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_set()
{
    last_command="devspace_set"

    command_aliases=()

    commands=()
    commands+=("analytics")
    commands+=("encryptionkey")
    commands+=("var")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_status_sync()
{
    last_command="devspace_status_sync"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_status()
{
    last_command="devspace_status"

    command_aliases=()

    commands=()
    commands+=("sync")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_sync()
{
    last_command="devspace_sync"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--container=")
    two_word_flags+=("--container")
    two_word_flags+=("-c")
    local_nonpersistent_flags+=("--container=")
    flags+=("--container-path=")
    two_word_flags+=("--container-path")
    local_nonpersistent_flags+=("--container-path=")
    flags+=("--download-on-initial-sync")
    local_nonpersistent_flags+=("--download-on-initial-sync")
    flags+=("--download-only")
    local_nonpersistent_flags+=("--download-only")
    flags+=("--exclude=")
    two_word_flags+=("--exclude")
    two_word_flags+=("-e")
    local_nonpersistent_flags+=("--exclude=")
    flags+=("--initial-sync=")
    two_word_flags+=("--initial-sync")
    local_nonpersistent_flags+=("--initial-sync=")
    flags+=("--label-selector=")
    two_word_flags+=("--label-selector")
    two_word_flags+=("-l")
    local_nonpersistent_flags+=("--label-selector=")
    flags+=("--local-path=")
    two_word_flags+=("--local-path")
    local_nonpersistent_flags+=("--local-path=")
    flags+=("--no-watch")
    local_nonpersistent_flags+=("--no-watch")
    flags+=("--pick")
    local_nonpersistent_flags+=("--pick")
    flags+=("--pod=")
    two_word_flags+=("--pod")
    local_nonpersistent_flags+=("--pod=")
    flags+=("--upload-only")
    local_nonpersistent_flags+=("--upload-only")
    flags+=("--verbose")
    local_nonpersistent_flags+=("--verbose")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_ui()
{
    last_command="devspace_ui"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--dev")
    local_nonpersistent_flags+=("--dev")
    flags+=("--port=")
    two_word_flags+=("--port")
    local_nonpersistent_flags+=("--port=")
    flags+=("--server")
    local_nonpersistent_flags+=("--server")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_update_config()
{
    last_command="devspace_update_config"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_update_dependencies()
{
    last_command="devspace_update_dependencies"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--allow-cyclic")
    local_nonpersistent_flags+=("--allow-cyclic")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_update()
{
    last_command="devspace_update"

    command_aliases=()

    commands=()
    commands+=("config")
    commands+=("dependencies")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_upgrade()
{
    last_command="devspace_upgrade"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_use_context()
{
    last_command="devspace_use_context"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_use_namespace()
{
    last_command="devspace_use_namespace"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--reset")
    local_nonpersistent_flags+=("--reset")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_use_profile()
{
    last_command="devspace_use_profile"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--reset")
    local_nonpersistent_flags+=("--reset")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_use_provider()
{
    last_command="devspace_use_provider"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_use_space()
{
    last_command="devspace_use_space"

    command_aliases=()

    commands=()

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--get-token")
    local_nonpersistent_flags+=("--get-token")
    flags+=("--provider=")
    two_word_flags+=("--provider")
    local_nonpersistent_flags+=("--provider=")
    flags+=("--space-id=")
    two_word_flags+=("--space-id")
    local_nonpersistent_flags+=("--space-id=")
    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_use()
{
    last_command="devspace_use"

    command_aliases=()

    commands=()
    commands+=("context")
    commands+=("namespace")
    commands+=("profile")
    commands+=("provider")
    commands+=("space")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

_devspace_root_command()
{
    last_command="devspace"

    command_aliases=()

    commands=()
    commands+=("add")
    commands+=("analyze")
    commands+=("attach")
    commands+=("build")
    commands+=("cleanup")
    commands+=("connect")
    commands+=("create")
    commands+=("deploy")
    commands+=("dev")
    commands+=("enter")
    commands+=("init")
    commands+=("list")
    commands+=("login")
    commands+=("logs")
    commands+=("open")
    commands+=("print")
    commands+=("purge")
    commands+=("remove")
    commands+=("render")
    commands+=("reset")
    commands+=("run")
    commands+=("set")
    commands+=("status")
    commands+=("sync")
    commands+=("ui")
    commands+=("update")
    commands+=("upgrade")
    commands+=("use")

    flags=()
    two_word_flags=()
    local_nonpersistent_flags=()
    flags_with_completion=()
    flags_completion=()

    flags+=("--config=")
    two_word_flags+=("--config")
    flags+=("--debug")
    flags+=("--kube-context=")
    two_word_flags+=("--kube-context")
    flags+=("--namespace=")
    two_word_flags+=("--namespace")
    two_word_flags+=("-n")
    flags+=("--no-warn")
    flags+=("--profile=")
    two_word_flags+=("--profile")
    two_word_flags+=("-p")
    flags+=("--silent")
    flags+=("--switch-context")
    flags+=("-s")
    flags+=("--var=")
    two_word_flags+=("--var")

    must_have_one_flag=()
    must_have_one_noun=()
    noun_aliases=()
}

__start_devspace()
{
    local cur prev words cword
    declare -A flaghash 2>/dev/null || :
    declare -A aliashash 2>/dev/null || :
    if declare -F _init_completion >/dev/null 2>&1; then
        _init_completion -s || return
    else
        __devspace_init_completion -n "=" || return
    fi

    local c=0
    local flags=()
    local two_word_flags=()
    local local_nonpersistent_flags=()
    local flags_with_completion=()
    local flags_completion=()
    local commands=("devspace")
    local must_have_one_flag=()
    local must_have_one_noun=()
    local has_completion_function
    local last_command
    local nouns=()

    __devspace_handle_word
}

if [[ $(type -t compopt) = "builtin" ]]; then
    complete -o default -F __start_devspace devspace
else
    complete -o default -o nospace -F __start_devspace devspace
fi

# ex: ts=4 sw=4 et filetype=sh
