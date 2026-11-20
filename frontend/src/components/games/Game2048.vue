<template>
  <div class="game-2048">
    <div class="game-header">
      <h2>2048</h2>
      <div class="score-board">
        <div class="score-item">
          <div class="score-label">分数</div>
          <div class="score-value">{{ score }}</div>
        </div>
        <div class="score-item">
          <div class="score-label">最高分</div>
          <div class="score-value">{{ bestScore }}</div>
        </div>
      </div>
      <el-button size="small" @click="restart">重新开始</el-button>
    </div>
    <div class="game-board" ref="gameBoard">
      <div class="grid-row" v-for="(row, i) in grid" :key="i">
        <div
          class="grid-cell"
          v-for="(cell, j) in row"
          :key="j"
          :class="`cell-${cell || 0}`"
        >
          {{ cell || '' }}
        </div>
      </div>
    </div>
    <div v-if="gameOver" class="game-over">
      <div class="game-over-content">
        <h3>游戏结束！</h3>
        <p>最终分数: {{ score }}</p>
        <el-button type="primary" @click="restart">再来一局</el-button>
      </div>
    </div>
    <div v-if="gameWon && !gameOver" class="game-won">
      <div class="game-won-content">
        <h3>恭喜！达到2048！</h3>
        <el-button type="primary" @click="continueGame">继续游戏</el-button>
        <el-button @click="restart">重新开始</el-button>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'Game2048',
  data() {
    return {
      grid: [],
      score: 0,
      bestScore: 0,
      gameOver: false,
      gameWon: false,
      size: 4
    };
  },
  mounted() {
    this.init();
    this.loadBestScore();
    this.bindKeyboard();
  },
  beforeUnmount() {
    this.unbindKeyboard();
  },
  methods: {
    init() {
      this.grid = Array(this.size).fill(null).map(() => Array(this.size).fill(0));
      this.score = 0;
      this.gameOver = false;
      this.gameWon = false;
      this.addRandomTile();
      this.addRandomTile();
    },
    loadBestScore() {
      const saved = localStorage.getItem('2048_best_score');
      if (saved) {
        this.bestScore = parseInt(saved, 10);
      }
    },
    saveBestScore() {
      if (this.score > this.bestScore) {
        this.bestScore = this.score;
        localStorage.setItem('2048_best_score', this.bestScore.toString());
      }
    },
    addRandomTile() {
      const emptyCells = [];
      for (let i = 0; i < this.size; i++) {
        for (let j = 0; j < this.size; j++) {
          if (this.grid[i][j] === 0) {
            emptyCells.push({ i, j });
          }
        }
      }
      if (emptyCells.length > 0) {
        const randomCell = emptyCells[Math.floor(Math.random() * emptyCells.length)];
        this.grid[randomCell.i][randomCell.j] = Math.random() < 0.9 ? 2 : 4;
      }
    },
    move(direction) {
      if (this.gameOver) return false;
      
      const prevGrid = JSON.parse(JSON.stringify(this.grid));
      let moved = false;
      
      if (direction === 'left') {
        moved = this.moveLeft();
      } else if (direction === 'right') {
        moved = this.moveRight();
      } else if (direction === 'up') {
        moved = this.moveUp();
      } else if (direction === 'down') {
        moved = this.moveDown();
      }
      
      if (moved) {
        this.addRandomTile();
        this.checkGameState();
        this.saveBestScore();
      }
      
      return moved;
    },
    moveLeft() {
      let moved = false;
      for (let i = 0; i < this.size; i++) {
        const row = this.grid[i].filter(cell => cell !== 0);
        const newRow = [];
        for (let j = 0; j < row.length; j++) {
          if (j < row.length - 1 && row[j] === row[j + 1]) {
            newRow.push(row[j] * 2);
            this.score += row[j] * 2;
            j++;
            moved = true;
          } else {
            newRow.push(row[j]);
            if (row[j] !== this.grid[i][j]) moved = true;
          }
        }
        while (newRow.length < this.size) {
          newRow.push(0);
        }
        this.grid[i] = newRow;
      }
      return moved;
    },
    moveRight() {
      let moved = false;
      for (let i = 0; i < this.size; i++) {
        const row = this.grid[i].filter(cell => cell !== 0);
        const newRow = [];
        for (let j = row.length - 1; j >= 0; j--) {
          if (j > 0 && row[j] === row[j - 1]) {
            newRow.unshift(row[j] * 2);
            this.score += row[j] * 2;
            j--;
            moved = true;
          } else {
            newRow.unshift(row[j]);
            if (row[j] !== this.grid[i][this.size - newRow.length]) moved = true;
          }
        }
        while (newRow.length < this.size) {
          newRow.unshift(0);
        }
        this.grid[i] = newRow;
      }
      return moved;
    },
    moveUp() {
      let moved = false;
      for (let j = 0; j < this.size; j++) {
        const column = [];
        for (let i = 0; i < this.size; i++) {
          if (this.grid[i][j] !== 0) {
            column.push(this.grid[i][j]);
          }
        }
        const newColumn = [];
        for (let i = 0; i < column.length; i++) {
          if (i < column.length - 1 && column[i] === column[i + 1]) {
            newColumn.push(column[i] * 2);
            this.score += column[i] * 2;
            i++;
            moved = true;
          } else {
            newColumn.push(column[i]);
            if (column[i] !== this.grid[newColumn.length - 1][j]) moved = true;
          }
        }
        while (newColumn.length < this.size) {
          newColumn.push(0);
        }
        for (let i = 0; i < this.size; i++) {
          if (this.grid[i][j] !== newColumn[i]) moved = true;
          this.grid[i][j] = newColumn[i];
        }
      }
      return moved;
    },
    moveDown() {
      let moved = false;
      for (let j = 0; j < this.size; j++) {
        const column = [];
        for (let i = 0; i < this.size; i++) {
          if (this.grid[i][j] !== 0) {
            column.push(this.grid[i][j]);
          }
        }
        const newColumn = [];
        for (let i = column.length - 1; i >= 0; i--) {
          if (i > 0 && column[i] === column[i - 1]) {
            newColumn.unshift(column[i] * 2);
            this.score += column[i] * 2;
            i--;
            moved = true;
          } else {
            newColumn.unshift(column[i]);
            if (column[i] !== this.grid[this.size - newColumn.length][j]) moved = true;
          }
        }
        while (newColumn.length < this.size) {
          newColumn.unshift(0);
        }
        for (let i = 0; i < this.size; i++) {
          if (this.grid[i][j] !== newColumn[i]) moved = true;
          this.grid[i][j] = newColumn[i];
        }
      }
      return moved;
    },
    checkGameState() {
      // 检查是否达到2048
      for (let i = 0; i < this.size; i++) {
        for (let j = 0; j < this.size; j++) {
          if (this.grid[i][j] === 2048 && !this.gameWon) {
            this.gameWon = true;
          }
        }
      }
      
      // 检查是否游戏结束
      let hasEmpty = false;
      let canMove = false;
      
      for (let i = 0; i < this.size; i++) {
        for (let j = 0; j < this.size; j++) {
          if (this.grid[i][j] === 0) {
            hasEmpty = true;
          }
          if (j < this.size - 1 && this.grid[i][j] === this.grid[i][j + 1]) {
            canMove = true;
          }
          if (i < this.size - 1 && this.grid[i][j] === this.grid[i + 1][j]) {
            canMove = true;
          }
        }
      }
      
      if (!hasEmpty && !canMove) {
        this.gameOver = true;
      }
    },
    restart() {
      this.init();
      this.saveBestScore();
    },
    continueGame() {
      this.gameWon = false;
    },
    bindKeyboard() {
      document.addEventListener('keydown', this.handleKeyPress);
    },
    unbindKeyboard() {
      document.removeEventListener('keydown', this.handleKeyPress);
    },
    handleKeyPress(e) {
      if (this.gameOver && !this.gameWon) return;
      
      const keyMap = {
        'ArrowLeft': 'left',
        'ArrowRight': 'right',
        'ArrowUp': 'up',
        'ArrowDown': 'down'
      };
      
      if (keyMap[e.key]) {
        e.preventDefault();
        this.move(keyMap[e.key]);
      }
    }
  }
};
</script>

<style scoped>
.game-2048 {
  padding: 20px;
  background: #faf8ef;
  border-radius: 8px;
  max-width: 500px;
  margin: 0 auto;
}

.game-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.game-header h2 {
  margin: 0;
  font-size: 48px;
  font-weight: bold;
  color: #776e65;
}

.score-board {
  display: flex;
  gap: 10px;
}

.score-item {
  background: #bbada0;
  padding: 8px 16px;
  border-radius: 4px;
  text-align: center;
  min-width: 80px;
}

.score-label {
  font-size: 12px;
  color: #eee4da;
  text-transform: uppercase;
  font-weight: bold;
}

.score-value {
  font-size: 20px;
  font-weight: bold;
  color: #fff;
}

.game-board {
  background: #bbada0;
  border-radius: 6px;
  padding: 10px;
  position: relative;
}

.grid-row {
  display: flex;
  gap: 10px;
  margin-bottom: 10px;
}

.grid-row:last-child {
  margin-bottom: 0;
}

.grid-cell {
  width: 100px;
  height: 100px;
  background: #cdc1b4;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 36px;
  font-weight: bold;
  color: #776e65;
  flex: 1;
}

.cell-2 { background: #eee4da; color: #776e65; }
.cell-4 { background: #ede0c8; color: #776e65; }
.cell-8 { background: #f2b179; color: #f9f6f2; }
.cell-16 { background: #f59563; color: #f9f6f2; }
.cell-32 { background: #f67c5f; color: #f9f6f2; }
.cell-64 { background: #f65e3b; color: #f9f6f2; }
.cell-128 { background: #edcf72; color: #f9f6f2; font-size: 32px; }
.cell-256 { background: #edcc61; color: #f9f6f2; font-size: 32px; }
.cell-512 { background: #edc850; color: #f9f6f2; font-size: 32px; }
.cell-1024 { background: #edc53f; color: #f9f6f2; font-size: 28px; }
.cell-2048 { background: #edc22e; color: #f9f6f2; font-size: 28px; }

.game-over,
.game-won {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(238, 228, 218, 0.9);
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
}

.game-over-content,
.game-won-content {
  text-align: center;
  padding: 20px;
}

.game-over-content h3,
.game-won-content h3 {
  margin: 0 0 10px 0;
  font-size: 32px;
  color: #776e65;
}

.game-over-content p {
  margin: 0 0 20px 0;
  font-size: 18px;
  color: #776e65;
}
</style>

