#!/bin/bash

die(){
    printf "something did not go as planned, %s\n" "$*"
    exit 7
}

printrun(){
    printf '\e[1;33mDEMO>\e[m '
    read -s
    printf '%s' "$*"
    read -s
    printf '\n'
    env -i "PATH=$base/bin:$PATH" "HOME=$base" bash -c "$*"
}

[[ $(head -n1 go.mod) == "module github.com/dedelala/disco" ]] || die "disco no go mod"
[[ -f disco.demo.yml ]] || die "disco.demo.yml not found"

base=$(mktemp -d /tmp/discoXXX) || die "no base"
trap "rm -rf $base" EXIT

mkdir -p "$base/bin" "$base/.config/" || die "mkdirs"
for cmd in disco discod printxy printgamut printlerp; do
    go build -o "$base/bin/$cmd" "./cmd/$cmd/." || die "build $cmd"
done
cp disco.demo.yml "$base/.config/disco.yml" || die "demo yml"
cd "$base" || die "cd base"

for c in $(shuf -e {196..230} | head -n 7); do
tput clear bold setaf "$c"

env -i "PATH=$base/bin:$PATH" "HOME=$base" bash -c "disco cue init-demo" || die "init-demo"

cat <<! >"$base/bin/demo.sh"
#!/bin/bash

printrun(){
    printf '\e[1;33mDEMO>\e[m '
    read -s
    printf '%s' "\$*"
    read -s
    printf '\n'
    bash -c "\$*"
}

printrun disco switch up1 on
printrun disco dim up1 25
printrun disco color up1 royal-blue

printrun disco switch up1
printrun disco dim up1
printrun disco color up1

printrun disco switch
printrun disco dim
printrun disco color

printrun disco switch downs on
printrun disco switch

!
chmod +x "$base/bin/demo.sh" || die "chmod demo.sh"

cat <<!
+-------------------------------------------------------------------------------+
|               ===========   =====  ==========   ========     ========         |
|               ===     ===   ===  ===      ==  ===    ===   ===    ===         |
|              ===      ===  ===  ====        ===          ===      ===         |
|             ===      ===  ===   ====       ===          ===      ===          |
|            ===      ===  ===     =====    ===          ===      ===           |
|           ===      ===  ===        ====  ===          ===      ===            |
|          ===      ===  ===         ==== ===          ===      ===             |
|         ===     ===   ===  ==      ===  ===    ===   ===    ===               |
|       ===========   ===== ==========    ========     ========                 |
+-------------------------------------------------------------------------------+
|               Domestic  Illumination  System  Control  Operator               |
+-------------------------------------------------------------------------------+
!

sleep 0.25
done

env -i "PATH=$base/bin:$PATH" "HOME=$base" 'PS1=\e[1;33mDEMO>\e[m ' bash
