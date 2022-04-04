# Copyright 1999-2022 Gentoo Authors
# Distributed under the terms of the GNU General Public License v2

EAPI=8

DESCRIPTION="Layercake binary-build-root manager"
HOMEPAGE="https://github.com/potano/layercake"
SRC_URI="https://github.com/potano/layercake/releases/download/${PV}/${P}.tar.gz"

LICENSE="GPL-2+"
SLOT="0"
KEYWORDS="amd64"
IUSE="doc"

BDEPEND=">=dev-lang/go-1.14.0
	doc? ( app-text/asciidoc app-text/docbook2X sys-apps/texinfo )"

src_install() {
	if use doc; then
		rm doc/*.{1,info}
	fi

	emake DESTDIR="${D}" install
}

