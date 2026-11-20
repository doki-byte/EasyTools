import { reactive } from 'vue'

// 分页工具函数
export default function usePagination(initialPageSize = 10) {
  // 使用 reactive 保证 table 的变更能触发视图更新
  const table = reactive({
    currentPage: 1,
    pageSize: initialPageSize,
    result: [],
    pageContent: [],
    selectRows: [],
    sortTemp: [],
    filterTemp: [],
    isSorted: false
  })

  const ctrl = {
    handleSizeChange(val) {
      table.pageSize = val
      table.currentPage = 1
      // 直接更新，slice操作很快
      if (table.isSorted) {
        table.pageContent = table.sortTemp.slice(0, val)
      } else {
        table.pageContent = table.result.slice(0, val)
      }
    },
    handleCurrentChange(val) {
      table.currentPage = val
      // 直接更新，slice操作很快
      const start = (val - 1) * table.pageSize
      const end = val * table.pageSize
      if (table.isSorted) {
        table.pageContent = table.sortTemp.slice(start, end)
      } else {
        table.pageContent = table.result.slice(start, end)
      }
    },
    handleSelectChange(rows) {
      table.selectRows = rows
    },
    watchResultChange(table) {
      const start = (table.currentPage - 1) * table.pageSize
      const end = table.currentPage * table.pageSize
      const source = table.isSorted ? table.sortTemp : table.result
      table.pageContent = source.slice(start, end)
    },
    sortChange(data, isVulTable = false) {
      if (!data.prop || !data.order) {
        table.isSorted = false
        ctrl.watchResultChange(table)
      } else {
        table.isSorted = true
        table.sortTemp = [...table.result]

        // 风险等级排序选项
        const sortSeverityOptions = ['CRITICAL', 'HIGH', 'MEDIUM', 'LOW', 'INFO', 'UNKNOWN']

        table.sortTemp.sort((a, b) => {
          let valA, valB
          if (isVulTable) {
            valA = sortSeverityOptions.indexOf(a['Severity'] || a['severity'] || 'UNKNOWN')
            valB = sortSeverityOptions.indexOf(b['Severity'] || b['severity'] || 'UNKNOWN')
          } else {
            valA = a[data.prop]
            valB = b[data.prop]
          }

          if (valA < valB) {
            return data.order === 'ascending' ? -1 : 1
          } else if (valA > valB) {
            return data.order === 'ascending' ? 1 : -1
          } else {
            return 0
          }
        })

        const start = (table.currentPage - 1) * table.pageSize
        const end = table.currentPage * table.pageSize
        table.pageContent = table.sortTemp.slice(start, end)
      }
    },
    getColumnFilters(prop) {
      const values = table.result.map(item => item[prop] || item[prop.toLowerCase()])
      const uniqueValues = Array.from(new Set(values))
      return uniqueValues.map(value => ({ text: value, value }))
    },
    filterChange(newFilters) {
      const filterKey = Object.keys(newFilters)[0]
      const selectedFilters = newFilters[filterKey] ? [...newFilters[filterKey]] : []

      if (table.filterTemp.length === 0) {
        table.filterTemp = [...table.result]
      }

      if (selectedFilters.length === 0) {
        table.result = [...table.filterTemp]
      } else {
        table.result = table.filterTemp.filter(item => {
          const itemValue = item[filterKey] || item[filterKey.toLowerCase()]
          return selectedFilters.includes(itemValue)
        })
      }
      ctrl.watchResultChange(table)
    },
    inputFilter(prop, value) {
      if (table.filterTemp.length === 0) {
        table.filterTemp = [...table.result]
      }
      if (value === '') {
        table.result = [...table.filterTemp]
      } else {
        table.result = table.filterTemp.filter(item => {
          const itemValue = String(item[prop] || item[prop.toLowerCase()] || '')
          return itemValue.toLowerCase().includes(value.toLowerCase())
        })
      }
      table.currentPage = 1
      ctrl.watchResultChange(table)
    }
  }

  const initTable = () => {
    table.result = []
    table.sortTemp = []
    table.filterTemp = []
    table.selectRows = []
    ctrl.watchResultChange(table)
  }

  return { table, ctrl, initTable }
}

