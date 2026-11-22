# Third-party notices

LAN Clipboard's own code is licensed under MIT. Dependencies and bundled
assets retain their original licenses.

- The Vivid file icons are from Daniel M. Hendricks' File Icon Vectors. See
  [the bundled MIT license and attribution](web/public/assets/file-icons/LICENSE.txt).
- The browser bundles use PDF.js and SheetJS Community Edition under
  Apache-2.0. Their upstream notices are included with the other frontend
  dependency licenses in the container.
- The Go binary includes the Go runtime and Go modules listed in `server/go.mod`,
  including the SQLite driver. SQLite itself is in the public domain.
- The container is based on Debian. Debian package notices are stored under
  `/usr/share/doc/` inside the image. The optional Gotenberg converter is
  distributed as a separate upstream image and uses its own licenses.

The container includes this notice, the project license and the license files
supplied by installed Go and frontend dependencies under
`/usr/share/licenses/lan-clipboard/`.
