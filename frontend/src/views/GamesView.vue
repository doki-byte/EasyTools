<template>
  <div class="games-view">
    <!-- 游戏选择界面 -->
    <div v-if="!currentGame" class="games-container">
      <div class="games-header">
        <h1>🎮 小游戏</h1>
        <p class="subtitle">放松一下，玩个小游戏吧！</p>
      </div>
      <div class="games-list">
        <div class="game-card" @click="selectGame('2048')">
          <div class="game-icon">2048</div>
          <div class="game-name">2048</div>
          <div class="game-desc">滑动合并数字，达到2048！</div>
          <div class="game-features">
            <span>🎯 策略</span>
            <span>🧠 益智</span>
          </div>
        </div>
        <div class="game-card" @click="selectGame('minesweeper')">
          <div class="game-icon">💣</div>
          <div class="game-name">扫雷</div>
          <div class="game-desc">找出所有地雷，挑战你的逻辑！</div>
          <div class="game-features">
            <span>🔍 推理</span>
            <span>⚡ 快速</span>
          </div>
        </div>
        <div class="game-card" @click="selectGame('sudoku')">
          <div class="game-icon">🔢</div>
          <div class="game-name">数独</div>
          <div class="game-desc">填入1-9，每行每列每宫不重复！</div>
          <div class="game-features">
            <span>🧩 逻辑</span>
            <span>🎯 专注</span>
          </div>
        </div>
      </div>
    </div>

    <!-- 游戏界面 -->
    <div v-else class="game-container">
      <div class="game-header-bar">
        <el-button @click="backToSelection" :icon="ArrowLeft">返回</el-button>
        <h2>{{ getGameTitle(currentGame) }}</h2>
        <div style="width: 80px;"></div>
      </div>
      <div class="game-content">
        <Game2048 v-if="currentGame === '2048'" />
        <Minesweeper v-else-if="currentGame === 'minesweeper'" />
        <Sudoku v-else-if="currentGame === 'sudoku'" />
      </div>
    </div>

  </div>
</template>

<script>
import Game2048 from '../components/games/Game2048.vue';
import Minesweeper from '../components/games/Minesweeper.vue';
import Sudoku from '../components/games/Sudoku.vue';
import { ArrowLeft } from '@element-plus/icons-vue';

export default {
  name: 'GamesView',
  components: {
    Game2048,
    Minesweeper,
    Sudoku
  },
  setup() {
    return {
      ArrowLeft
    };
  },
  data() {
    return {
      currentGame: null
    };
  },
  mounted() {
    // 游戏视图已加载
  },
  beforeUnmount() {
    // 清理工作
  },
  methods: {
    selectGame(game) {
      this.currentGame = game;
    },
    backToSelection() {
      this.currentGame = null;
    },
    getGameTitle(game) {
      const titles = {
        '2048': '2048游戏',
        'minesweeper': '扫雷游戏',
        'sudoku': '数独游戏'
      };
      return titles[game] || '游戏';
    }
  }
};
</script>

<style scoped>
.games-view {
  height: 100%;
  padding: 20px;
  background-color: #9ab4db;
  overflow-y: auto;
}

.games-container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 40px 20px;
}

.games-header {
  text-align: center;
  margin-bottom: 50px;
  color: white;
}

.games-header h1 {
  margin: 0 0 15px 0;
  font-size: 48px;
  font-weight: bold;
  text-shadow: 2px 2px 4px rgba(0, 0, 0, 0.2);
}

.subtitle {
  margin: 0;
  font-size: 18px;
  opacity: 0.95;
  text-shadow: 1px 1px 2px rgba(0, 0, 0, 0.2);
}

.games-list {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 30px;
  margin-top: 40px;
}

.game-card {
  background: rgba(255, 255, 255, 0.2);
  backdrop-filter: blur(10px);
  border-radius: 16px;
  padding: 40px;
  text-align: center;
  cursor: pointer;
  transition: all 0.3s ease;
  border: 2px solid transparent;
  color: white;
}

.game-card:hover {
  background: rgba(255, 255, 255, 0.3);
  transform: translateY(-8px);
  border-color: rgba(255, 255, 255, 0.5);
  box-shadow: 0 15px 40px rgba(0, 0, 0, 0.3);
}

.game-icon {
  font-size: 64px;
  margin-bottom: 20px;
  font-weight: bold;
}

.game-name {
  font-size: 28px;
  font-weight: bold;
  margin-bottom: 15px;
}

.game-desc {
  font-size: 16px;
  opacity: 0.95;
  line-height: 1.6;
  margin-bottom: 20px;
}

.game-features {
  display: flex;
  justify-content: center;
  gap: 15px;
  flex-wrap: wrap;
}

.game-features span {
  background: rgba(255, 255, 255, 0.2);
  padding: 6px 12px;
  border-radius: 20px;
  font-size: 14px;
  font-weight: 500;
}

.game-container {
  max-width: 1000px;
  margin: 0 auto;
  padding: 20px;
}

.game-header-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  background: rgba(255, 255, 255, 0.95);
  padding: 15px 20px;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.game-header-bar h2 {
  margin: 0;
  font-size: 24px;
  color: #333;
}

.game-content {
  background: rgba(255, 255, 255, 0.95);
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

/* 响应式设计 */
@media (max-width: 768px) {
  .games-header h1 {
    font-size: 36px;
  }
  
  .games-list {
    grid-template-columns: 1fr;
  }
  
  .game-card {
    padding: 30px 20px;
  }
}
</style>

