package resolve

import "strings"

// Common browser-side libraries that are loaded by <script> tags, not imported: calls to them are external.
// A project function with the same name still wins (candidates are tried before builtins).
func init() {
  for _, w := range strings.Fields("jsPDF Chart Swal Toastify flatpickr html2canvas jQuery axios moment dayjs Papa XLSX Sortable") {
    builtins[w] = true
  }
}
