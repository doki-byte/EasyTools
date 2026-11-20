<template>
  <div class="minesweeper">
    <div class="game-header">
      <h2>扫雷</h2>
      <div class="game-info">
        <div class="info-item">
          <span class="info-label">剩余雷数:</span>
          <span class="info-value">{{ remainingMines }}</span>
        </div>
        <div class="info-item">
          <span class="info-label">用时:</span>
          <span class="info-value">{{ formatTime(timeElapsed) }}</span>
        </div>
      </div>
      <el-button size="small" @click="restart">重新开始</el-button>
    </div>
    <div class="game-board" :style="{ gridTemplateColumns: `repeat(${cols}, 1fr)` }">
      <div
        v-for="(cell, index) in cells"
        :key="index"
        class="cell"
        :class="{
          'revealed': cell.revealed,
          'flagged': cell.flagged,
          'mine': cell.revealed && cell.isMine,
          'exploded': cell.exploded
        }"
        @click="handleCellClick(cell, index)"
        @contextmenu.prevent="handleRightClick(cell, index)"
      >
        <span v-if="cell.revealed && !cell.isMine && cell.neighborMines > 0" 
              :class="`number-${cell.neighborMines}`">
          {{ cell.neighborMines }}
        </span>
        <span v-else-if="cell.flagged">🚩</span>
        <span v-else-if="cell.revealed && cell.isMine">💣</span>
      </div>
    </div>
    <div v-if="gameOver" class="game-overlay">
      <div class="game-over-content">
        <h3>{{ gameWon ? '恭喜！你赢了！' : '游戏结束！' }}</h3>
        <p v-if="gameWon">用时: {{ formatTime(timeElapsed) }}</p>
        <el-button type="primary" @click="restart">再来一局</el-button>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'Minesweeper',
  props: {
    rows: {
      type: Number,
      default: 16
    },
    cols: {
      type: Number,
      default: 16
    },
    mines: {
      type: Number,
      default: 40
    }
  },
  data() {
    return {
      cells: [],
      gameOver: false,
      gameWon: false,
      gameStarted: false,
      timeElapsed: 0,
      timer: null,
      remainingMines: 40
    };
  },
  mounted() {
    this.init();
  },
  beforeUnmount() {
    this.clearTimer();
  },
  methods: {
    init() {
      this.cells = [];
      this.gameOver = false;
      this.gameWon = false;
      this.gameStarted = false;
      this.timeElapsed = 0;
      this.remainingMines = this.mines;
      this.clearTimer();
      
      // 创建所有单元格
      for (let i = 0; i < this.rows * this.cols; i++) {
        this.cells.push({
          isMine: false,
          revealed: false,
          flagged: false,
          neighborMines: 0,
          exploded: false
        });
      }
    },
    placeMines(excludeIndex) {
      let placed = 0;
      while (placed < this.mines) {
        const index = Math.floor(Math.random() * this.rows * this.cols);
        if (index !== excludeIndex && !this.cells[index].isMine) {
          this.cells[index].isMine = true;
          placed++;
        }
      }
      
      // 计算每个单元格周围的雷数
      for (let i = 0; i < this.cells.length; i++) {
        this.cells[i].neighborMines = this.countNeighborMines(i);
      }
    },
    countNeighborMines(index) {
      const row = Math.floor(index / this.cols);
      const col = index % this.cols;
      let count = 0;
      
      for (let i = -1; i <= 1; i++) {
        for (let j = -1; j <= 1; j++) {
          if (i === 0 && j === 0) continue;
          const newRow = row + i;
          const newCol = col + j;
          if (newRow >= 0 && newRow < this.rows && newCol >= 0 && newCol < this.cols) {
            const neighborIndex = newRow * this.cols + newCol;
            if (this.cells[neighborIndex].isMine) {
              count++;
            }
          }
        }
      }
      return count;
    },
    handleCellClick(cell, index) {
      if (this.gameOver || cell.flagged || cell.revealed) return;
      
      if (!this.gameStarted) {
        this.gameStarted = true;
        this.placeMines(index);
        this.startTimer();
      }
      
      if (cell.isMine) {
        cell.exploded = true;
        this.endGame(false);
        return;
      }
      
      this.revealCell(index);
      this.checkWin();
    },
    handleRightClick(cell, index) {
      if (this.gameOver || cell.revealed) return;
      
      if (cell.flagged) {
        cell.flagged = false;
        this.remainingMines++;
      } else {
        cell.flagged = true;
        this.remainingMines--;
      }
      this.checkWin();
    },
    revealCell(index) {
      const cell = this.cells[index];
      if (cell.revealed || cell.flagged) return;
      
      cell.revealed = true;
      
      // 如果周围没有雷，自动展开周围的单元格
      if (cell.neighborMines === 0) {
        const row = Math.floor(index / this.cols);
        const col = index % this.cols;
        
        for (let i = -1; i <= 1; i++) {
          for (let j = -1; j <= 1; j++) {
            if (i === 0 && j === 0) continue;
            const newRow = row + i;
            const newCol = col + j;
            if (newRow >= 0 && newRow < this.rows && newCol >= 0 && newCol < this.cols) {
              const neighborIndex = newRow * this.cols + newCol;
              if (!this.cells[neighborIndex].revealed && !this.cells[neighborIndex].flagged) {
                this.revealCell(neighborIndex);
              }
            }
          }
        }
      }
    },
    checkWin() {
      let revealedCount = 0;
      for (const cell of this.cells) {
        if (cell.revealed) {
          revealedCount++;
        }
      }
      
      if (revealedCount === this.rows * this.cols - this.mines) {
        this.endGame(true);
      }
    },
    endGame(won) {
      this.gameOver = true;
      this.gameWon = won;
      this.clearTimer();
      
      if (!won) {
        // 显示所有雷
        for (const cell of this.cells) {
          if (cell.isMine && !cell.exploded) {
            cell.revealed = true;
          }
        }
      }
    },
    restart() {
      this.init();
    },
    startTimer() {
      this.timer = setInterval(() => {
        this.timeElapsed++;
      }, 1000);
    },
    clearTimer() {
      if (this.timer) {
        clearInterval(this.timer);
        this.timer = null;
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
.minesweeper {
  padding: 20px;
  background: #f5f5f5;
  border-radius: 8px;
  max-width: 600px;
  margin: 0 auto;
  position: relative;
}

.game-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  flex-wrap: wrap;
  gap: 10px;
}

.game-header h2 {
  margin: 0;
  font-size: 32px;
  color: #333;
}

.game-info {
  display: flex;
  gap: 20px;
}

.info-item {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.info-label {
  font-size: 12px;
  color: #666;
  margin-bottom: 4px;
}

.info-value {
  font-size: 20px;
  font-weight: bold;
  color: #333;
}

.game-board {
  display: grid;
  gap: 2px;
  background: #ccc;
  padding: 2px;
  border-radius: 4px;
  position: relative;
}

.cell {
  aspect-ratio: 1;
  background: #ddd;
  border: 2px outset #ddd;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  font-weight: bold;
  cursor: pointer;
  user-select: none;
  min-width: 30px;
  min-height: 30px;
}

.cell:hover:not(.revealed):not(.flagged) {
  background: #e0e0e0;
}

.cell.revealed {
  background: #f0f0f0;
  border: 1px solid #ccc;
  cursor: default;
}

.cell.flagged {
  background: #ddd;
}

.cell.mine {
  background: #ff4444;
}

.cell.exploded {
  background: #ff0000;
  animation: explode 0.3s;
}

@keyframes explode {
  0% { transform: scale(1); }
  50% { transform: scale(1.2); }
  100% { transform: scale(1); }
}

.number-1 { color: #0000ff; }
.number-2 { color: #008000; }
.number-3 { color: #ff0000; }
.number-4 { color: #000080; }
.number-5 { color: #800000; }
.number-6 { color: #008080; }
.number-7 { color: #000000; }
.number-8 { color: #808080; }

.game-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  z-index: 10;
}

.game-over-content {
  background: white;
  padding: 30px;
  border-radius: 8px;
  text-align: center;
}

.game-over-content h3 {
  margin: 0 0 10px 0;
  font-size: 24px;
  color: #333;
}

.game-over-content p {
  margin: 0 0 20px 0;
  font-size: 16px;
  color: #666;
}
</style>

