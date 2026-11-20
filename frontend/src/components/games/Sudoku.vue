<template>
  <div class="sudoku-container">
    <div class="sudoku-header">
      <div class="game-info">
        <div class="info-item">
          <span class="label">难度：</span>
          <el-select v-model="difficulty" size="small" @change="newGame" style="width: 100px;">
            <el-option label="简单" value="easy" />
            <el-option label="中等" value="medium" />
            <el-option label="困难" value="hard" />
          </el-select>
        </div>
        <div class="info-item">
          <span class="label">时间：</span>
          <span class="value">{{ formatTime(timer) }}</span>
        </div>
        <div class="info-item">
          <span class="label">错误：</span>
          <span class="value">{{ mistakes }}</span>
        </div>
      </div>
      <div class="game-actions">
        <el-button size="small" @click="newGame" :icon="Refresh">新游戏</el-button>
        <el-button size="small" @click="hint" :disabled="!canHint" :icon="InfoFilled">提示</el-button>
        <el-button size="small" @click="checkSolution" :icon="Check">检查</el-button>
        <el-button size="small" @click="solve" :icon="MagicStick">求解</el-button>
      </div>
    </div>

    <div class="sudoku-main-content">
      <div class="sudoku-board-wrapper">
        <div class="sudoku-board" :class="{ 'solved': isSolved }">
          <div
            v-for="(row, rowIndex) in board"
            :key="rowIndex"
            class="sudoku-row"
            :class="{ 'row-border-bottom': (rowIndex + 1) % 3 === 0 && rowIndex < 8 }"
          >
            <div
              v-for="(cell, colIndex) in row"
              :key="colIndex"
              class="sudoku-cell"
              :class="{
                'cell-border-right': (colIndex + 1) % 3 === 0 && colIndex < 8,
                'initial': cell.initial,
                'selected': selectedCell && selectedCell.row === rowIndex && selectedCell.col === colIndex,
                'error': cell.error,
                'hint': cell.hint
              }"
              @click="selectCell(rowIndex, colIndex)"
            >
              <input
                v-if="!cell.initial"
                v-model.number="cell.value"
                type="number"
                min="1"
                max="9"
                class="cell-input"
                @input="onCellInput(rowIndex, colIndex, $event)"
                @keydown="onCellKeydown($event, rowIndex, colIndex)"
              />
              <span v-else class="cell-initial">{{ cell.value || '' }}</span>
              <div v-if="cell.notes && cell.notes.length > 0" class="cell-notes">
                <span v-for="note in cell.notes" :key="note" class="note">{{ note }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div class="sudoku-controls-panel">
        <div class="number-pad">
          <div class="number-pad-title">选择数字</div>
          <div class="number-pad-buttons">
            <button
              v-for="num in 9"
              :key="num"
              class="number-btn"
              :class="{ 'active': selectedCell && getCellValue(selectedCell.row, selectedCell.col) === num }"
              @click="setCellValue(num)"
            >
              {{ num }}
            </button>
            <button class="number-btn clear-btn" @click="clearCell">清除</button>
          </div>
        </div>
        <div v-if="isSolved" class="solved-message">
          <el-alert type="success" :closable="false" show-icon>
            <template #title>恭喜！你完成了数独！</template>
          </el-alert>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { Refresh, InfoFilled, Check, MagicStick } from '@element-plus/icons-vue';

export default {
  name: 'Sudoku',
  components: {},
  setup() {
    return {
      Refresh,
      InfoFilled,
      Check,
      MagicStick
    };
  },
  data() {
    return {
      board: [],
      solution: [],
      difficulty: 'medium',
      selectedCell: null,
      mistakes: 0,
      timer: 0,
      timerInterval: null,
      isSolved: false,
      hintsUsed: 0,
      maxHints: 3
    };
  },
  computed: {
    canHint() {
      return this.hintsUsed < this.maxHints && this.selectedCell && !this.isSolved;
    }
  },
  mounted() {
    this.newGame();
    this.startTimer();
  },
  beforeUnmount() {
    this.stopTimer();
  },
  methods: {
    newGame() {
      this.isSolved = false;
      this.mistakes = 0;
      this.hintsUsed = 0;
      this.timer = 0;
      this.selectedCell = null;
      this.generateSudoku();
      this.startTimer();
    },
    generateSudoku() {
      // 生成完整解
      this.solution = this.createEmptyBoard();
      this.solveSudoku(this.solution);
      
      // 创建游戏板（挖空）
      this.board = this.solution.map(row => 
        row.map(value => ({
          value: value,
          initial: true,
          error: false,
          hint: false,
          notes: []
        }))
      );
      
      // 根据难度移除数字
      const removeCount = {
        easy: 30,
        medium: 45,
        hard: 55
      };
      
      let removed = 0;
      const targetRemove = removeCount[this.difficulty];
      
      while (removed < targetRemove) {
        const row = Math.floor(Math.random() * 9);
        const col = Math.floor(Math.random() * 9);
        
        if (this.board[row][col].initial) {
          this.board[row][col].value = 0;
          this.board[row][col].initial = false;
          removed++;
        }
      }
    },
    createEmptyBoard() {
      return Array(9).fill(null).map(() => Array(9).fill(0));
    },
    solveSudoku(board) {
      // 使用回溯算法生成数独解
      const numbers = [1, 2, 3, 4, 5, 6, 7, 8, 9];
      
      // 打乱数字顺序以增加随机性
      for (let i = numbers.length - 1; i > 0; i--) {
        const j = Math.floor(Math.random() * (i + 1));
        [numbers[i], numbers[j]] = [numbers[j], numbers[i]];
      }
      
      const isValid = (row, col, num) => {
        // 检查行
        for (let c = 0; c < 9; c++) {
          if (board[row][c] === num) return false;
        }
        
        // 检查列
        for (let r = 0; r < 9; r++) {
          if (board[r][col] === num) return false;
        }
        
        // 检查3x3宫格
        const startRow = Math.floor(row / 3) * 3;
        const startCol = Math.floor(col / 3) * 3;
        for (let r = startRow; r < startRow + 3; r++) {
          for (let c = startCol; c < startCol + 3; c++) {
            if (board[r][c] === num) return false;
          }
        }
        
        return true;
      };
      
      const solve = () => {
        for (let row = 0; row < 9; row++) {
          for (let col = 0; col < 9; col++) {
            if (board[row][col] === 0) {
              for (const num of numbers) {
                if (isValid(row, col, num)) {
                  board[row][col] = num;
                  if (solve()) {
                    return true;
                  }
                  board[row][col] = 0;
                }
              }
              return false;
            }
          }
        }
        return true;
      };
      
      solve();
    },
    selectCell(row, col) {
      if (!this.board[row][col].initial) {
        this.selectedCell = { row, col };
        // 清除之前的错误标记
        this.clearErrors();
      }
    },
    getCellValue(row, col) {
      return this.board[row][col]?.value || 0;
    },
    setCellValue(value) {
      if (!this.selectedCell || this.isSolved) return;
      
      const { row, col } = this.selectedCell;
      if (this.board[row][col].initial) return;
      
      this.board[row][col].value = value;
      this.board[row][col].error = false;
      this.board[row][col].hint = false;
      
      // 验证输入
      this.validateCell(row, col);
      
      // 检查是否完成
      this.checkCompletion();
      
      // 移动到下一个单元格
      this.moveToNextCell();
    },
    clearCell() {
      if (!this.selectedCell || this.isSolved) return;
      
      const { row, col } = this.selectedCell;
      if (this.board[row][col].initial) return;
      
      this.board[row][col].value = 0;
      this.board[row][col].error = false;
      this.clearErrors();
    },
    onCellInput(row, col, event) {
      const value = parseInt(event.target.value) || 0;
      if (value >= 1 && value <= 9) {
        this.board[row][col].value = value;
        this.validateCell(row, col);
        this.checkCompletion();
      } else if (value === 0) {
        this.board[row][col].value = 0;
        this.clearErrors();
      } else {
        event.target.value = this.board[row][col].value || '';
      }
    },
    onCellKeydown(event, row, col) {
      const { key } = event;
      
      if (key === 'ArrowUp' && row > 0) {
        event.preventDefault();
        this.selectCell(row - 1, col);
      } else if (key === 'ArrowDown' && row < 8) {
        event.preventDefault();
        this.selectCell(row + 1, col);
      } else if (key === 'ArrowLeft' && col > 0) {
        event.preventDefault();
        this.selectCell(row, col - 1);
      } else if (key === 'ArrowRight' && col < 8) {
        event.preventDefault();
        this.selectCell(row, col + 1);
      } else if (key === 'Delete' || key === 'Backspace') {
        event.preventDefault();
        this.clearCell();
      } else if (key >= '1' && key <= '9') {
        // 数字键已在input中处理
      }
    },
    moveToNextCell() {
      if (!this.selectedCell) return;
      
      let { row, col } = this.selectedCell;
      col++;
      if (col >= 9) {
        col = 0;
        row++;
        if (row >= 9) {
          row = 0;
        }
      }
      
      // 跳过初始单元格
      while (row < 9 && this.board[row][col].initial) {
        col++;
        if (col >= 9) {
          col = 0;
          row++;
        }
      }
      
      if (row < 9) {
        this.selectCell(row, col);
        // 聚焦输入框
        this.$nextTick(() => {
          const cell = document.querySelector(`.sudoku-row:nth-child(${row + 1}) .sudoku-cell:nth-child(${col + 1}) .cell-input`);
          if (cell) cell.focus();
        });
      }
    },
    validateCell(row, col) {
      const value = this.board[row][col].value;
      if (!value) {
        this.board[row][col].error = false;
        return;
      }
      
      // 检查行
      for (let c = 0; c < 9; c++) {
        if (c !== col && this.board[row][c].value === value) {
          this.board[row][col].error = true;
          this.mistakes++;
          return;
        }
      }
      
      // 检查列
      for (let r = 0; r < 9; r++) {
        if (r !== row && this.board[r][col].value === value) {
          this.board[row][col].error = true;
          this.mistakes++;
          return;
        }
      }
      
      // 检查3x3宫格
      const startRow = Math.floor(row / 3) * 3;
      const startCol = Math.floor(col / 3) * 3;
      for (let r = startRow; r < startRow + 3; r++) {
        for (let c = startCol; c < startCol + 3; c++) {
          if ((r !== row || c !== col) && this.board[r][c].value === value) {
            this.board[row][col].error = true;
            this.mistakes++;
            return;
          }
        }
      }
      
      // 检查是否正确
      if (this.solution[row][col] === value) {
        this.board[row][col].error = false;
      }
    },
    clearErrors() {
      for (let row = 0; row < 9; row++) {
        for (let col = 0; col < 9; col++) {
          this.board[row][col].error = false;
        }
      }
    },
    checkCompletion() {
      for (let row = 0; row < 9; row++) {
        for (let col = 0; col < 9; col++) {
          if (!this.board[row][col].value || this.board[row][col].value !== this.solution[row][col]) {
            return;
          }
        }
      }
      
      // 所有单元格都正确
      this.isSolved = true;
      this.stopTimer();
      this.$message.success('恭喜！你完成了数独！');
    },
    checkSolution() {
      let hasError = false;
      this.clearErrors();
      
      for (let row = 0; row < 9; row++) {
        for (let col = 0; col < 9; col++) {
          const value = this.board[row][col].value;
          if (value && value !== this.solution[row][col]) {
            this.board[row][col].error = true;
            hasError = true;
          }
        }
      }
      
      if (!hasError) {
        this.$message.success('所有数字都正确！');
      } else {
        this.$message.warning('发现一些错误，请检查红色标记的单元格');
      }
    },
    hint() {
      if (!this.canHint || !this.selectedCell) return;
      
      const { row, col } = this.selectedCell;
      if (this.board[row][col].initial || this.board[row][col].value) return;
      
      // 显示正确答案
      this.board[row][col].value = this.solution[row][col];
      this.board[row][col].hint = true;
      this.hintsUsed++;
      
      this.$message.info(`提示已使用 ${this.hintsUsed}/${this.maxHints}`);
      this.checkCompletion();
    },
    solve() {
      this.$confirm('确定要自动求解吗？这将显示完整答案。', '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }).then(() => {
        for (let row = 0; row < 9; row++) {
          for (let col = 0; col < 9; col++) {
            if (!this.board[row][col].initial) {
              this.board[row][col].value = this.solution[row][col];
              this.board[row][col].hint = false;
              this.board[row][col].error = false;
            }
          }
        }
        this.isSolved = true;
        this.stopTimer();
        this.$message.success('已自动求解');
      }).catch(() => {});
    },
    startTimer() {
      this.stopTimer();
      this.timerInterval = setInterval(() => {
        this.timer++;
      }, 1000);
    },
    stopTimer() {
      if (this.timerInterval) {
        clearInterval(this.timerInterval);
        this.timerInterval = null;
      }
    },
    formatTime(seconds) {
      const mins = Math.floor(seconds / 60);
      const secs = seconds % 60;
      return `${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
    }
  }
};
</script>

<style scoped>
.sudoku-container {
  display: flex;
  flex-direction: column;
  padding: 20px;
  gap: 20px;
}

.sudoku-header {
  width: 100%;
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 15px;
}

.game-info {
  display: flex;
  gap: 20px;
  align-items: center;
  flex-wrap: wrap;
}

.info-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.label {
  font-weight: 500;
  color: #666;
}

.value {
  font-weight: 600;
  color: #333;
  font-size: 16px;
}

.game-actions {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

.sudoku-main-content {
  display: flex;
  gap: 30px;
  align-items: flex-start;
  justify-content: center;
  flex-wrap: wrap;
}

.sudoku-board-wrapper {
  display: flex;
  justify-content: center;
  padding: 20px;
  background: #f5f5f5;
  border-radius: 8px;
  flex-shrink: 0;
}

.sudoku-board {
  display: inline-block;
  background: white;
  border: 3px solid #333;
  border-radius: 4px;
  padding: 4px;
}

.sudoku-board.solved {
  border-color: #67c23a;
  box-shadow: 0 0 20px rgba(103, 194, 58, 0.3);
}

.sudoku-row {
  display: flex;
}

.sudoku-row.row-border-bottom {
  border-bottom: 3px solid #333;
}

.sudoku-cell {
  width: 50px;
  height: 50px;
  border: 1px solid #ddd;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  background: white;
  cursor: pointer;
  transition: all 0.2s;
}

.sudoku-cell.cell-border-right {
  border-right: 3px solid #333;
}

.sudoku-cell:hover:not(.initial) {
  background: #f0f9ff;
}

.sudoku-cell.selected {
  background: #e1f5ff !important;
  border-color: #409eff;
  box-shadow: inset 0 0 0 2px #409eff;
}

.sudoku-cell.initial {
  background: #f5f5f5;
  font-weight: bold;
  color: #333;
  cursor: default;
}

.sudoku-cell.error {
  background: #fef0f0 !important;
  color: #f56c6c;
}

.sudoku-cell.hint {
  background: #f0f9ff !important;
  color: #409eff;
  font-weight: 600;
}

.cell-input {
  width: 100%;
  height: 100%;
  border: none;
  text-align: center;
  font-size: 20px;
  font-weight: 500;
  background: transparent;
  outline: none;
  color: #333;
}

.cell-input::-webkit-inner-spin-button,
.cell-input::-webkit-outer-spin-button {
  -webkit-appearance: none;
  margin: 0;
}

.cell-input[type=number] {
  -moz-appearance: textfield;
}

.cell-initial {
  font-size: 20px;
  font-weight: bold;
  color: #333;
}

.cell-notes {
  position: absolute;
  top: 2px;
  left: 2px;
  font-size: 8px;
  color: #999;
  display: flex;
  flex-wrap: wrap;
  gap: 1px;
}

.note {
  display: inline-block;
  width: 8px;
  height: 8px;
  line-height: 8px;
  text-align: center;
}

.sudoku-controls-panel {
  display: flex;
  flex-direction: column;
  gap: 20px;
  min-width: 200px;
}

.number-pad {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 20px;
  background: #f5f5f5;
  border-radius: 8px;
}

.number-pad-title {
  font-size: 16px;
  font-weight: 600;
  color: #333;
  text-align: center;
  margin-bottom: 8px;
}

.number-pad-buttons {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 10px;
}

.number-btn {
  height: 55px;
  border: 2px solid #ddd;
  border-radius: 8px;
  background: white;
  font-size: 22px;
  font-weight: 600;
  color: #333;
  cursor: pointer;
  transition: all 0.2s;
  min-width: 55px;
}

.number-btn:hover {
  background: #f0f9ff;
  border-color: #409eff;
  transform: translateY(-2px);
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.1);
}

.number-btn.active {
  background: #409eff;
  color: white;
  border-color: #409eff;
}

.clear-btn {
  grid-column: span 3;
  background: #f56c6c;
  color: white;
  border-color: #f56c6c;
  margin-top: 5px;
}

.clear-btn:hover {
  background: #f78989;
  border-color: #f78989;
}

.solved-message {
  width: 100%;
  max-width: 500px;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .sudoku-main-content {
    flex-direction: column;
    align-items: center;
  }
  
  .sudoku-controls-panel {
    width: 100%;
    max-width: 300px;
  }
  
  .sudoku-cell {
    width: 40px;
    height: 40px;
  }
  
  .cell-input,
  .cell-initial {
    font-size: 18px;
  }
  
  .sudoku-header {
    flex-direction: column;
    align-items: flex-start;
  }
  
  .game-info {
    width: 100%;
  }
  
  .game-actions {
    width: 100%;
    justify-content: flex-start;
  }
}

@media (max-width: 480px) {
  .sudoku-cell {
    width: 35px;
    height: 35px;
  }
  
  .cell-input,
  .cell-initial {
    font-size: 16px;
  }
}
</style>

